package logger

import (
	"encoding/json"
	"io"
	sync "sync"

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
	LogEntries     int
	InvalidEntries int

	LoginAttempt      LoginAttemptReport
	RunCommand        RunCommandReport
	UnknownCommand    UnknownCommandReport
	InvalidInvocation InvalidInvocationReport
	Credentials       CredentialsReport
	Download          DownloadReport
	Panic             PanicReport
}

func (r *Report) Update(le *LogEntry) {
	r.LogEntries++

	switch event := le.GetLogType().(type) {
	case *LogEntry_LoginAttempt:
		r.LoginAttempt.update(event.LoginAttempt)
	default:
		r.InvalidEntries++
	}
}

type LoginAttemptReport struct {
	// List of passwords and their counts.
	Passwords StrCounter
	// List of usernames and their counts.
	Usernames StrCounter
	// List of login attempt results and their counts.
	Results StrCounter
}

func (r *LoginAttemptReport) update(la *LoginAttempt) {
	r.Passwords.Increment(la.Password)
	r.Usernames.Increment(la.Username)
	r.Results.Increment(la.GetResult().String())
}

type RunCommandReport struct {
}

type UnknownCommandReport struct {
}

type InvalidInvocationReport struct {
}

type CredentialsReport struct {
}

type DownloadReport struct {
}

type PanicReport struct {
}

type StrCounter struct {
	internal map[string]int
	init     sync.Once
}

func (s *StrCounter) Increment(toAdd string) {
	s.init.Do(func() {
		s.internal = make(map[string]int)
	})
	s.internal[toAdd]++
}
