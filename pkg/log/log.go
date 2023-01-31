/*
 * Copyright 2022 The DIAMBRA Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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

func New() *Logger {
	return &Logger{Logger: log.With(log.NewLogfmtLogger(os.Stderr), "caller", log.Caller(4))}
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
	l.Logger = log.With(l.Logger, "caller", log.Caller(4))
	l.Logger = log.With(l.Logger, "source", "cli")

	return nil
}
