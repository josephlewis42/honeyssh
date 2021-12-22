package logger

import (
	"encoding/json"
	"fmt"
	"io"

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
