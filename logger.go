package tinyflags

import (
	"context"
	"fmt"
	"log"
	"os"
)

type Logger interface {
	Errorf(ctx context.Context, format string, v ...any)
}

type defaultLogger struct {
	log *log.Logger
}

func (l *defaultLogger) Errorf(_ context.Context, format string, v ...any) {
	_ = l.log.Output(2, fmt.Sprintf(format, v...))
}

var logger Logger = &defaultLogger{
	log: log.New(os.Stderr, "tinyflags: ", log.LstdFlags|log.Lshortfile),
}

func SetLogger(l Logger) {
	logger = l
}
