package ttylog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	sync "sync"
	"time"
)

// AsciicastFileExt holds the suggested file extension for asciicast files.
const AsciicastFileExt = "cast"

func writeJSONLine(w io.Writer, structure interface{}) error {
	line, err := json.Marshal(structure)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "%s\n", string(line))
	return err
}

// NewAsciicastLogSink creates a LogSink compatible with the asciicast v2
// format.
//
// See: https://github.com/asciinema/asciinema/blob/develop/doc/asciicast-v2.md
func NewAsciicastLogSink(w io.Writer) LogSink {
	var (
		firstLogTimeMicros int64
		once               sync.Once
	)

	return func(entry *TTYLogEntry) error {
		var headerErr error
		once.Do(func() {
			firstLogTimeMicros = entry.GetTimestampMicros()
			// Give generic settings that should work to display most outputs.
			headerErr = writeJSONLine(w, map[string]interface{}{
				"version":   2,
				"width":     80,
				"height":    24,
				"timestamp": time.UnixMicro(firstLogTimeMicros).Unix(),
				"title":     "github.com/josephlewis42/honeyssh session",
				"env": map[string]interface{}{
					"TERM":  "xterm-256color",
					"SHELL": "/bin/sh",
				},
			})
		})
		if headerErr != nil {
			return headerErr
		}

		deltaSecond := microsecondsToSeconds(entry.GetTimestampMicros() - firstLogTimeMicros)

		switch event := entry.GetEvent().(type) {
		case *TTYLogEntry_Io:
			direction := "o"
			if event.Io.GetFd() == FD_STDIN {
				direction = "i"
			}
			data := string(event.Io.Data)

			return writeJSONLine(w, &asciicastLogLine{deltaSecond, direction, data})
		case *TTYLogEntry_Close:
			// No-op.
			return nil
		default:
			return fmt.Errorf("unknown event: %T", entry.GetEvent())
		}
	}
}

type AsciicastLogSource struct {
	r             *bufio.Reader
	consumeHeader sync.Once
}

var _ LogSource = (*AsciicastLogSource)(nil)

// NewAsciicastLogSource reads log events from an Asciicast formatted file.
func NewAsciicastLogSource(r io.Reader) *AsciicastLogSource {
	return &AsciicastLogSource{r: bufio.NewReader(r)}
}

// Next gets the next log entry, it returns io.EOF if there are no more.
func (log *AsciicastLogSource) Next() (*TTYLogEntry, error) {
	log.consumeHeader.Do(func() {
		log.r.ReadBytes('\n')
	})

	for {
		line, err := log.r.ReadBytes('\n')
		if err != nil {
			return nil, err
		}

		if len(line) == 1 {
			// Skip blank lines
			continue
		}

		var asciicastLine asciicastLogLine
		if err := json.Unmarshal(line, &asciicastLine); err != nil {
			return nil, err
		}

		// Asciicast doesn't support stderr so it's collapsed into stdout.
		var fd FD
		switch asciicastLine.EventType {
		case "o":
			fd = FD_STDOUT
		case "i":
			fd = FD_STDIN
		default:
			// skip unknown events
			continue
		}

		return &TTYLogEntry{
			TimestampMicros: secondsToMicroseconds(asciicastLine.TimeSeconds),
			Event: &TTYLogEntry_Io{
				Io: &IO{
					Data: []byte(asciicastLine.EventData),
					Fd:   fd,
				},
			},
		}, nil
	}
}

type asciicastLogLine struct {
	TimeSeconds float64
	EventType   string
	EventData   string
}

func (log *asciicastLogLine) UnmarshalJSON(data []byte) error {
	var v []interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	if count := len(v); count != 3 {
		return fmt.Errorf("malformed line, expected 3 entries got %d", count)
	}

	var timeOk, typeOk, dataOk bool
	log.TimeSeconds, timeOk = v[0].(float64)
	log.EventType, typeOk = v[1].(string)
	log.EventData, dataOk = v[2].(string)

	if !timeOk || !typeOk || !dataOk {
		return fmt.Errorf("malformed data in line: %q", v)
	}

	return nil
}

func (log *asciicastLogLine) MarshalJSON() ([]byte, error) {
	data := string(log.EventData)

	return json.Marshal([]interface{}{log.TimeSeconds, log.EventType, data})

}

func microsecondsToSeconds(microseconds int64) (seconds float64) {
	return (float64(microseconds) * float64(time.Microsecond)) / float64(time.Second)
}

func secondsToMicroseconds(seconds float64) (microseconds int64) {
	return int64(float64(seconds)*float64(time.Second)) / int64(time.Microsecond)
}
