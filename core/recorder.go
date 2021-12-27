package core

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"sync"
	"time"

	"josephlewis.net/honeyssh/core/vos"
)

type MockFd int

const (
	fdStdin  MockFd = 0
	fdStdout        = 1
	fdStderr        = 2
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

type Recorder struct {
	*vos.VIOAdapter
	mutex  sync.Mutex
	output io.Writer
}

type event struct {
	Operation    int32  // Operation, maps into MockFdOp.
	Tty          uint32 // Should always be 0.
	Size         int32  // Number of bytes following this event that represent the data.
	Direction    int32  // Data direction, maps into MockFdDir.
	Seconds      uint32 // UNIX timestamp of the event.
	Microseconds uint32 // Microseconds after the timestamp of the event.
}

// According to Kippo, the format matches User Mode Linux recording.
func logEvent(out io.Writer, timestamp time.Time, mockFd MockFd, op MockFdOp, data []byte) error {
	sec := timestamp.UnixNano() / int64(time.Second)
	usec := (timestamp.UnixNano() % int64(time.Second)) / int64(time.Microsecond)

	direction := dirWrite
	if mockFd == fdStdin {
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

func (r *Recorder) recordRead(mockFd MockFd, from io.Reader, to []byte) (int, error) {
	amount, err := from.Read(to)
	if err == nil {
		readTime := time.Now()
		r.mutex.Lock()
		e2 := logEvent(r.output, readTime, mockFd, opWrite, to[:amount])
		r.mutex.Unlock()
		if e2 != nil {
			log.Print(e2)
		}
	}
	return amount, err
}

func (r *Recorder) recordWrite(mockFd MockFd, from []byte, to io.Writer) (int, error) {
	writeTime := time.Now()
	amount, err := to.Write(from)
	if err == nil {
		r.mutex.Lock()
		e2 := logEvent(r.output, writeTime, mockFd, opWrite, from[:amount])
		r.mutex.Unlock()
		if e2 != nil {
			log.Print(e2)
		}
	}

	return amount, err
}

var _ vos.VIO = (*Recorder)(nil)

type recorderReadCloser struct {
	r       *Recorder
	mockFd  MockFd
	wrapped io.ReadCloser
}

var _ io.ReadCloser = (*recorderReadCloser)(nil)

func (rc *recorderReadCloser) Read(p []byte) (int, error) {
	return rc.r.recordRead(rc.mockFd, rc.wrapped, p)
}

func (rc *recorderReadCloser) Close() error {
	return rc.wrapped.Close()
}

type recorderWriteCloser struct {
	r       *Recorder
	mockFd  MockFd
	wrapped io.WriteCloser
}

var _ io.WriteCloser = (*recorderWriteCloser)(nil)

func (rc *recorderWriteCloser) Write(p []byte) (int, error) {
	return rc.r.recordWrite(rc.mockFd, p, rc.wrapped)
}

func (rc *recorderWriteCloser) Close() error {
	return rc.wrapped.Close()
}

// Record logs all events to output.
func Record(toWrap vos.VIO, output io.Writer) *Recorder {
	recorder := &Recorder{
		output: output,
	}

	recorder.VIOAdapter = vos.NewVIOAdapter(
		&recorderReadCloser{mockFd: fdStdin, r: recorder, wrapped: toWrap.Stdin()},
		&recorderWriteCloser{mockFd: fdStdout, r: recorder, wrapped: toWrap.Stdout()},
		&recorderWriteCloser{mockFd: fdStderr, r: recorder, wrapped: toWrap.Stderr()},
	)

	return recorder
}

type replayOpts struct {
	maxSleep time.Duration
}

// ReplayOpt changes options for playback
type ReplayOpt func(*replayOpts)

// MaxSleep sets the maximum duration that Replay will sleep when playing
// events.
func MaxSleep(duration time.Duration) ReplayOpt {
	return func(r *replayOpts) {
		r.maxSleep = duration
	}
}

// Replay plays a stream of events to destination.
func Replay(recording io.Reader, destination io.Writer, opts ...ReplayOpt) (err error) {
	options := &replayOpts{
		maxSleep: 3 * time.Second,
	}

	for _, o := range opts {
		o(options)
	}

	var prevTime time.Time
	var once sync.Once
	eventPtr := &event{}
	buf := &bytes.Buffer{}

	for {
		if err := binary.Read(recording, binary.LittleEndian, eventPtr); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		buf.Reset()

		currTime := time.Unix(int64(eventPtr.Seconds), int64(eventPtr.Microseconds)*int64(time.Microsecond))
		once.Do(func() {
			prevTime = currTime
		})
		if _, err := io.CopyN(buf, recording, int64(eventPtr.Size)); err != nil {
			return err
		}

		switch {
		case MockFdOp(eventPtr.Operation) == opClose:
			continue

		case MockFdOp(eventPtr.Operation) == opWrite:
			sleepDuration := currTime.Sub(prevTime)
			if sleepDuration > options.maxSleep {
				sleepDuration = options.maxSleep
			}
			time.Sleep(sleepDuration)

			if MockFdDir(eventPtr.Direction) == dirWrite {
				if _, err := destination.Write(buf.Bytes()); err != nil {
					return err
				}
			}
		}

		prevTime = currTime
	}
}

// EventType is the type of event that the LogEvent represents.
type EventType int

const (
	EventTypeClose EventType = iota
	EventTypeInput
	EventTypeOutput
)

type LogEvent struct {
	// Timestamp of this event.
	Time time.Time
	// Type of the event.
	EventType EventType
	// Data associated with the event.
	Data []byte
}

// ReplayCallback reads a stream of events to a callback.
func ReplayCallback(recording io.Reader, callback func(*LogEvent) error) (err error) {
	eventPtr := &event{}
	buf := &bytes.Buffer{}

	for {
		if err := binary.Read(recording, binary.LittleEndian, eventPtr); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		buf.Reset()

		currTime := time.Unix(int64(eventPtr.Seconds), int64(eventPtr.Microseconds)*int64(time.Microsecond))
		if _, err := io.CopyN(buf, recording, int64(eventPtr.Size)); err != nil {
			return err
		}

		outputEvent := &LogEvent{
			Time: currTime,
			Data: buf.Bytes(),
		}

		switch {
		case MockFdOp(eventPtr.Operation) == opClose:
			outputEvent.EventType = EventTypeClose
			if err := callback(outputEvent); err != nil {
				return err
			}

		case MockFdOp(eventPtr.Operation) == opWrite:
			switch {
			case MockFdDir(eventPtr.Direction) == dirWrite:
				outputEvent.EventType = EventTypeOutput
			default:
				outputEvent.EventType = EventTypeInput
			}
			if err := callback(outputEvent); err != nil {
				return err
			}
		}
	}
}
