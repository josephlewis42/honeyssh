package ttylog

import (
	"io"
	"log"
	"regexp"
	sync "sync"
	"time"

	"github.com/josephlewis42/honeyssh/core/vos"
)

var (
	crlf = regexp.MustCompile(`\r?\n`)
)

// LogSink receives log events.
type LogSink func(t *TTYLogEntry) error

// LogSource adapts log readers.
type LogSource interface {
	// Next fetches the next available log entry. It reutrns io.EOF if the source
	// has no more log entries.
	Next() (*TTYLogEntry, error)
}

// NewRealTimePlayback plays back the results in real-time.
// If maxSleep > 0, it's used as the maximum duration to pause.
func NewRealTimePlayback(maxSleep time.Duration, next LogSink) LogSink {
	var once sync.Once
	var prevTimeMicros int64

	return func(logEntry *TTYLogEntry) error {
		once.Do(func() {
			prevTimeMicros = logEntry.GetTimestampMicros()
		})

		delta := logEntry.GetTimestampMicros() - prevTimeMicros
		prevTimeMicros = logEntry.GetTimestampMicros()

		if maxSleep > 0 {
			sleepDuration := time.Duration(delta) * time.Microsecond
			if sleepDuration > maxSleep {
				sleepDuration = maxSleep
			}
			time.Sleep(sleepDuration)
		}

		return next(logEntry)
	}
}

// NewKippoQuirksAdapter fixes quirks in log events that come from Kippo.
func NewKippoQuirksAdapter(next LogSink) LogSink {
	return func(logEntry *TTYLogEntry) error {
		// Kippo sent \n rather than \r\n meaning that playback could creep across
		// the screen because the cursor position wasn't reset.
		if event, ok := logEntry.GetEvent().(*TTYLogEntry_Io); ok {
			event.Io.Data = crlf.ReplaceAll(event.Io.GetData(), []byte("\r\n"))
		}

		return next(logEntry)
	}
}

// NewClientOutput writes stdout and stderr to the given writer
func NewClientOutput(w io.Writer) LogSink {
	return func(logEntry *TTYLogEntry) error {
		if event, ok := logEntry.Event.(*TTYLogEntry_Io); ok {
			if event.Io.GetFd() != FD_STDIN {
				_, err := w.Write(event.Io.GetData())
				if err != nil {
					return err
				}
			}
		}
		return nil
	}
}

// Replay reads a stream of events to a callback.
func Replay(recording LogSource, callback LogSink) (err error) {
	for {
		logEntry, err := recording.Next()
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		}

		if err := callback(logEntry); err != nil {
			return err
		}
	}
}

type Recorder struct {
	*vos.VIOAdapter
	mutex  sync.Mutex
	output LogSink
}

func (r *Recorder) recordIO(mockFd FD, data []byte, dest func([]byte) (int, error)) (int, error) {
	eventTime := time.Now()
	amount, err := dest(data)
	if err == nil {
		r.mutex.Lock()
		e2 := r.output(&TTYLogEntry{
			TimestampMicros: eventTime.UnixMicro(),
			Event: &TTYLogEntry_Io{
				Io: &IO{
					Fd:   mockFd,
					Data: data,
				},
			},
		})
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
	mockFd  FD
	wrapped io.ReadCloser
}

var _ io.ReadCloser = (*recorderReadCloser)(nil)

func (rc *recorderReadCloser) Read(p []byte) (int, error) {
	return rc.r.recordIO(rc.mockFd, p, rc.wrapped.Read)
}

func (rc *recorderReadCloser) Close() error {
	return rc.wrapped.Close()
}

type recorderWriteCloser struct {
	r       *Recorder
	mockFd  FD
	wrapped io.WriteCloser
}

var _ io.WriteCloser = (*recorderWriteCloser)(nil)

func (rc *recorderWriteCloser) Write(p []byte) (int, error) {
	return rc.r.recordIO(rc.mockFd, p, rc.wrapped.Write)
}

func (rc *recorderWriteCloser) Close() error {
	return rc.wrapped.Close()
}

// NewRecorder creates a logger that forwards all events to output.
func NewRecorder(toWrap vos.VIO, output LogSink) *Recorder {
	recorder := &Recorder{
		output: output,
	}

	recorder.VIOAdapter = vos.NewVIOAdapter(
		&recorderReadCloser{mockFd: FD_STDIN, r: recorder, wrapped: toWrap.Stdin()},
		&recorderWriteCloser{mockFd: FD_STDOUT, r: recorder, wrapped: toWrap.Stdout()},
		&recorderWriteCloser{mockFd: FD_STDERR, r: recorder, wrapped: toWrap.Stderr()},
	)

	return recorder
}
