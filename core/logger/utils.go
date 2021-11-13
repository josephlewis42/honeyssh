package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"time"
)

type session struct{}

var sessionMarker = session{}

func StartSession(ctx context.Context) context.Context {
	return context.WithValue(ctx, sessionMarker, fmt.Sprintf("%d", rand.Uint64()))
}

func GetSessionId(ctx context.Context) string {
	if v := ctx.Value(sessionMarker); v != nil {
		return v.(string)
	}

	return ""
}

type LogRecorder func(le *LogEntry) error

type Logger struct {
	Record LogRecorder
}

func NewJsonLinesLogRecorder(w io.Writer) *Logger {
	return &Logger{
		Record: func(le *LogEntry) error {
			entry, err := json.Marshal(le)
			if err != nil {
				return err
			}
			_, err = fmt.Println(string(entry))
			return err
		},
	}
}

func (l *Logger) recordLogType(ctx context.Context, resource string, event isLogEntry_LogType) error {
	le := &LogEntry{}
	le.TimestampMicros = time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
	le.SessionId = GetSessionId(ctx)
	le.Resource = resource
	le.LogType = event

	return l.Record(le)
}

func (l *Logger) RecordLoginAttempt(ctx context.Context, resource string, event *LoginAttempt) error {
	return l.recordLogType(ctx, resource, &LogEntry_LoginAttempt{
		LoginAttempt: event,
	})
}
