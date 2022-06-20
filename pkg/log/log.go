package log

import (
	"fmt"
	"os"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type Logger struct {
	log.Logger
}

func (l *Logger) SetOptions(debug bool, logFormat string) error {
	switch logFormat {
	case "logfmt":
		l.Logger = log.NewLogfmtLogger(os.Stderr)
	case "json":
		l.Logger = log.NewJSONLogger(os.Stderr)
	case "fancy":
		l.Logger = NewFancyLogger(os.Stderr)
	default:
		return fmt.Errorf("invalid log format %s", logFormat)
	}
	if !debug {
		l.Logger = level.NewFilter(l.Logger, level.AllowInfo())
	}
	l.Logger = log.With(l.Logger, "caller", log.Caller(5))
	l.Logger = log.With(l.Logger, "source", "cli")

	return nil
}
