package ttylog

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

type MockFdOp int

const (
	opOpen  MockFdOp = 1
	opClose          = 2
	opWrite          = 3
	opExec           = 4
)

type MockFdDir int

const (
	dirRead  MockFdDir = 1
	dirWrite MockFdDir = 2
)

type event struct {
	Operation    int32  // Operation, maps into MockFdOp.
	Tty          uint32 // Should always be 0.
	Size         int32  // Number of bytes following this event that represent the data.
	Direction    int32  // Data direction, maps into MockFdDir.
	Seconds      uint32 // UNIX timestamp of the event.
	Microseconds uint32 // Microseconds after the timestamp of the event.
}

// According to Kippo, the format matches User Mode Linux recording.
func logEvent(out io.Writer, timestamp time.Time, mockFd FD, op MockFdOp, data []byte) error {
	sec := timestamp.UnixNano() / int64(time.Second)
	usec := (timestamp.UnixNano() % int64(time.Second)) / int64(time.Microsecond)

	direction := dirWrite
	if mockFd == FD_STDIN {
		direction = dirRead
	}

	eventData := []interface{}{
		int32(op),
		uint32(0), // TTY, always 0
		int32(len(data)),
		int32(direction),
		uint32(sec),
		uint32(usec),
	}

	for _, v := range eventData {
		err := binary.Write(out, binary.LittleEndian, v)
		if err != nil {
			return err
		}
	}

	if len(data) > 0 {
		if _, err := out.Write(data); err != nil {
			return err
		}
	}

	return nil
}

// NewUMLLogSink creates a LogSink compatible with the user-mode-linux TTY.
func NewUMLLogSink(w io.Writer) LogSink {
	return func(entry *TTYLogEntry) error {
		timestamp := time.UnixMicro(entry.TimestampMicros)

		switch event := entry.GetEvent().(type) {
		case *TTYLogEntry_Io:
			return logEvent(w, timestamp, event.Io.Fd, opWrite, event.Io.Data)
		case *TTYLogEntry_Close:
			return logEvent(w, timestamp, event.Close.Fd, opClose, nil)
		default:
			return fmt.Errorf("unknown event: %T", entry.GetEvent())
		}
	}
}

// UMLLogSource parses log events from a user-mode-linux/Kippo formatted file.
type UMLLogSource struct {
	r io.Reader
}

var _ LogSource = (*UMLLogSource)(nil)

// NewUMLLogSource reads log events from a user-mode-linux/Kippo formatted file.
func NewUMLLogSource(r io.Reader) *UMLLogSource {
	return &UMLLogSource{r: r}
}

// Next gets the next log entry, it returns io.EOF if there are no more.
func (log *UMLLogSource) Next() (*TTYLogEntry, error) {
	eventPtr := &event{}
	buf := &bytes.Buffer{}

	for {
		// Read the event's data
		if err := binary.Read(log.r, binary.LittleEndian, eventPtr); err != nil {
			return nil, io.EOF
		}
		buf.Reset()
		if _, err := io.CopyN(buf, log.r, int64(eventPtr.Size)); err != nil {
			return nil, err
		}

		// Extract the event details

		logTime := (int64(eventPtr.Seconds) * int64(time.Second)) / int64(time.Microsecond)
		logTime += int64(eventPtr.Microseconds)

		// UML doesn't distinguish between stdout and stderr so we'll report it all
		// as stdout.
		var fd FD = FD_STDOUT
		if MockFdDir(eventPtr.Direction) == dirRead {
			fd = FD_STDIN
		}

		switch MockFdOp(eventPtr.Operation) {
		case opClose:
			return &TTYLogEntry{
				TimestampMicros: logTime,
				Event: &TTYLogEntry_Close{
					Close: &Close{
						Fd: fd,
					},
				},
			}, nil
		case opWrite:
			return &TTYLogEntry{
				TimestampMicros: logTime,
				Event: &TTYLogEntry_Io{
					Io: &IO{
						Data: buf.Bytes(),
						Fd:   fd,
					},
				},
			}, nil
		case opOpen, opExec:
			fallthrough
		default:
			// Skip unknown or non-I/O operations
			continue
		}
	}
}
