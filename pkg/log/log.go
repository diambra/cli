package log

import (
	"os"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type Logger struct {
	log.Logger
}

func (l *Logger) SetLogLevel(lo level.Option) {
	l.Logger = level.NewFilter(l.Logger, lo)
}

func New() *Logger {
	logger := log.NewLogfmtLogger(os.Stderr)
	logger = log.With(logger, "caller", log.Caller(5))

	return &Logger{logger}
}
