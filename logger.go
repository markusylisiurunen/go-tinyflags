package tinyflags

import (
	"context"
	"fmt"
	"log"
	"os"
)

const debug = false

type Logger interface {
	Errorf(ctx context.Context, format string, v ...any)
}

type debugLogger interface {
	Logger
	Debugf(format string, v ...any)
}

type wrappedLogger struct {
	Logger
}

func (l *wrappedLogger) Debugf(format string, v ...any) {}

type defaultLogger struct {
	log *log.Logger
}

func (l *defaultLogger) Errorf(_ context.Context, format string, v ...any) {
	_ = l.log.Output(2, fmt.Sprintf(format, v...))
}

func (l *defaultLogger) Debugf(format string, v ...any) {
	if !debug {
		return
	}
	_ = l.log.Output(2, fmt.Sprintf(format, v...))
}

var logger debugLogger = &defaultLogger{
	log: log.New(os.Stderr, "tinyflags: ", log.LstdFlags|log.Lshortfile),
}

func SetLogger(l Logger) {
	logger = &wrappedLogger{l}
}
