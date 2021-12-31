package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
)

// ReadJSONLinesLog parses a newline delimited JSON log.
func ReadJSONLinesLog(r io.Reader, handler func(le *LogEntry)) error {
	decoder := json.NewDecoder(r)
	for decoder.More() {
		var rawEntry json.RawMessage
		if err := decoder.Decode(&rawEntry); err != nil {
			return err
		}

		var logEntry LogEntry
		if err := protojson.Unmarshal(rawEntry, &logEntry); err != nil {
			return err
		}

		handler(&logEntry)
	}
	return nil
}

func NewBugReport() *BugReport {
	return &BugReport{
		InvalidInvocations: NewPathCounter("command", "error"),
		UnknownCommands:    NewPathCounter("command", "status", "error"),
	}
}

// BugReport pulls events that are likely bugs in the honeypot.
type BugReport struct {
	LogEntries int

	InvalidInvocations *PathCounter `json:"invalid_invocations"`
	UnknownCommands    *PathCounter `json:"unknown_commands"`
	Panics             []*Panic     `json:"panics"`
}

func (r *BugReport) Update(le *LogEntry) {
	r.LogEntries++

	switch event := le.GetLogType().(type) {
	case *LogEntry_Panic:
		r.Panics = append(r.Panics, event.Panic)
	case *LogEntry_UnknownCommand:
		msg := event.UnknownCommand
		r.UnknownCommands.Increment(msg.Command[0], msg.GetStatus().String(), msg.ErrorMessage)
	case *LogEntry_InvalidInvocation:
		msg := event.InvalidInvocation
		r.InvalidInvocations.Increment(msg.Command[0], msg.Error)
	}
}

type InteractionReport struct {
	// Map of sessionID -> interactions
	interactions map[string]*InteractiveSession
}

type InteractiveSession struct {
	Login struct {
		Username   string `json:"username"`
		Password   string `json:"password"`
		PublicKey  []byte `json:"public_key,omitempty"`
		RemoteAddr string `json:"remote_addr,omitempty"`
	} `json:"login"`
	TTYLog       string `json:"tty_log"`
	LogEntries   int    `json:"log_entries"`
	TerminalName string `json:"terminal_name"`
	IsPty        bool   `json:"is_pty"`

	Commands  []string `json:"commands"`
	Downloads []string `json:"downloads"`
}

func (i *InteractiveSession) Update(le *LogEntry) {
	i.LogEntries++

	switch event := le.GetLogType().(type) {
	case *LogEntry_LoginAttempt:
		i.Login.Password = event.LoginAttempt.GetPassword()
		i.Login.Username = event.LoginAttempt.GetUsername()
		i.Login.PublicKey = event.LoginAttempt.GetPublicKey()
		i.Login.RemoteAddr = event.LoginAttempt.GetRemoteAddr()
	case *LogEntry_RunCommand:
		i.Commands = append(i.Commands, strings.Join(event.RunCommand.GetCommand(), " "))
	case *LogEntry_Download:
		i.Downloads = append(i.Downloads, fmt.Sprintf("%q -> %q", event.Download.GetSource(), event.Download.GetName()))
	case *LogEntry_UnknownCommand:
		i.Commands = append(i.Commands, strings.Join(event.UnknownCommand.GetCommand(), " "))
	case *LogEntry_TerminalUpdate:
		i.TerminalName = event.TerminalUpdate.GetTerm()
		i.IsPty = event.TerminalUpdate.GetIsPty()
	case *LogEntry_OpenTtyLog:
		i.TTYLog = event.OpenTtyLog.GetName()
	}
}

func (i *InteractionReport) init() {
	if i.interactions == nil {
		i.interactions = make(map[string]*InteractiveSession)
	}
}

// MarshalJSON implemnts custom JSON marshaler.
func (i *InteractionReport) MarshalJSON() ([]byte, error) {
	i.init()

	return json.Marshal(i.interactions)
}

func (i *InteractionReport) Update(le *LogEntry) {
	i.init()

	sessionID := le.GetSessionId()
	if sessionID == "" {
		return
	}
	report, ok := i.interactions[sessionID]
	if !ok {
		report = &InteractiveSession{}
		i.interactions[sessionID] = report
	}

	report.Update(le)
}

// Report holds statistics about the logged events.
type Report struct {
	LogEntries     int        `json:"log_entries"`
	InvalidEntries StrCounter `json:"unknown_log_entries,omitempty"`

	LoginAttempt      LoginAttemptReport      `json:"login_attempt_report"`
	RunCommand        RunCommandReport        `json:"run_command_report"`
	UnknownCommand    UnknownCommandReport    `json:"unknown_command_report"`
	InvalidInvocation InvalidInvocationReport `json:"invalid_invocation_report"`
	Credentials       CredentialsReport       `json:"credential_report"`
	Download          DownloadReport          `json:"download_report"`
	Panic             PanicReport             `json:"panic_report"`
}

func (r *Report) Update(le *LogEntry) {
	r.LogEntries++

	switch event := le.GetLogType().(type) {
	case *LogEntry_LoginAttempt:
		r.LoginAttempt.update(event.LoginAttempt)
	case *LogEntry_RunCommand:
		r.RunCommand.update(event.RunCommand)
	case *LogEntry_Panic:
		r.Panic.update(event.Panic)
	case *LogEntry_Download:
		r.Download.update(event.Download)
	case *LogEntry_UnknownCommand:
		r.UnknownCommand.update(event.UnknownCommand)
	case *LogEntry_InvalidInvocation:
		r.InvalidInvocation.update(event.InvalidInvocation)
	case *LogEntry_TerminalUpdate, *LogEntry_HoneypotEvent, *LogEntry_OpenTtyLog:
		// Ignore
	default:
		r.InvalidEntries.Increment(fmt.Sprintf("%T", event))
	}
}

type LoginAttemptReport struct {
	// List of passwords and their counts.
	Passwords StrCounter `json:"passwords"`
	// List of usernames and their counts.
	Usernames StrCounter `json:"usernames"`
	// List of login attempt results and their counts.
	Results StrCounter `json:"results"`
}

func (r *LoginAttemptReport) update(la *LoginAttempt) {
	r.Passwords.Increment(la.Password)
	r.Usernames.Increment(la.Username)
	r.Results.Increment(la.GetResult().String())
}

type RunCommandReport struct {
	// Name of the resolved command
	ResolvedCommandPaths StrCounter `json:"resolved_command_names"`
	// Name of the command
	CommandNames StrCounter `json:"command_names"`
}

func (r *RunCommandReport) update(rc *RunCommand) {
	r.ResolvedCommandPaths.Increment(rc.ResolvedCommandPath)
	if len(rc.Command) > 0 {
		r.CommandNames.Increment(rc.Command[0])
	}
}

type UnknownCommandReport struct {
	CommandNames    StrCounter `json:"command_names"`
	CommandStatuses StrCounter `json:"command_statuses"`
}

func (r *UnknownCommandReport) update(logEntry *UnknownCommand) {
	if len(logEntry.Command) > 0 {
		r.CommandNames.Increment(logEntry.Command[0])
	}

	r.CommandStatuses.Increment(logEntry.Status.String())
}

type InvalidInvocationReport struct {
	CommandNames StrCounter `json:"command_counts"`
}

func (r *InvalidInvocationReport) update(logEntry *InvalidInvocation) {
	if len(logEntry.Command) > 0 {
		r.CommandNames.Increment(logEntry.Command[0])
	}
}

type CredentialsReport struct {
}

type DownloadReport struct {
	Count        int        `json:"count"`
	Sources      StrCounter `json:"sources"`
	CommandNames StrCounter `json:"command_counts"`
}

func (r *DownloadReport) update(d *Download) {
	r.Count++
	r.Sources.Increment(d.Source)
	if len(d.Command) > 0 {
		r.CommandNames.Increment(d.Command[0])
	}
}

type PanicReport struct {
	Contexts []string `json:"contexts"`
}

func (r *PanicReport) update(p *Panic) {
	r.Contexts = append(r.Contexts, p.Context)
}

// StrCounter counts the number of strings seen.
type StrCounter struct {
	internal map[string]int
}

// Increment adds one to the given key.
func (s *StrCounter) Increment(toAdd string) {
	if s.internal == nil {
		s.internal = make(map[string]int)
	}

	s.internal[toAdd]++
}

// MarshalJSON implemnts custom JSON marshaler.
func (s StrCounter) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.internal)
}

func NewPathCounter(cols ...string) *PathCounter {
	return &PathCounter{
		cols:     cols,
		internal: make(map[string]int),
	}
}

// PathCounter counts the number of strings seen.
type PathCounter struct {
	cols     []string
	internal map[string]int
}

// Increment adds one to the given key.
func (ctr *PathCounter) Increment(toAdd ...string) {
	if len(toAdd) != len(ctr.cols) {
		panic("wrong number of columns to add")
	}

	ctr.internal[toKey(toAdd...)]++
}

// MarshalJSON implemnts custom JSON marshaler.
func (ctr *PathCounter) MarshalJSON() ([]byte, error) {
	type Count struct {
		Count  int               `json:"count"`
		Fields map[string]string `json:"event"`
		Path   string            `json:"-"`
	}

	var out []Count
	for k, v := range ctr.internal {
		count := Count{
			Count:  v,
			Path:   k,
			Fields: make(map[string]string),
		}

		splitPath := fromKey(k)
		for colNum, colVal := range ctr.cols {
			count.Fields[colVal] = splitPath[colNum]
		}

		out = append(out, count)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Path < out[j].Path
		}
		return out[i].Count > out[j].Count
	})

	return json.Marshal(out)
}

func toKey(vals ...string) string {
	key, _ := json.Marshal(vals)
	return string(key)
}

func fromKey(key string) (out []string) {
	json.Unmarshal([]byte(key), &out)
	return
}
