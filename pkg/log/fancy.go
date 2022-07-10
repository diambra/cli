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
	"io"

	"github.com/fatih/color"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

func NewFancyLogger(w io.Writer) log.Logger {
	return &fancyLogger{w}
}

func (l *fancyLogger) Log(kvs ...interface{}) error {
	// Make sure we have a equal number of keys/values
	if len(kvs)%2 == 1 {
		kvs = append(kvs, nil)
	}

	var (
		msg    string
		icon   string
		id     string
		errstr string
		col    = color.New(color.Reset)
	)
	for i := 0; i < len(kvs); i += 2 {
		key, val := kvs[i], kvs[i+1]
		switch k := key.(type) {
		case string:
			switch k {
			case "err":
				errstr = ": " + val.(string)
			case "msg":
				msg = val.(string)
			case "id":
				id = fmt.Sprintf("(%s) ", val.(string)[:4])
			case "source":
				switch val.(string) {
				case "agent":
					icon = "🤖 "
				case "env":
					icon = "🏟️ "
				case "cli":
					icon = "🖥️  "
				}
			case "level":
				switch val.(level.Value).String() {
				case "warn":
					col = color.New(color.FgYellow)
				case "error":
					col = color.New(color.FgRed)
				case "info":
					col = color.New(color.FgHiWhite)
				case "debug":
					col = color.New(color.FgHiBlack)
				}
			}
		}
	}

	col.Fprintf(l.w, "\033[0G\033[2K%s%s%s%s\n", icon, id, msg, errstr)
	return nil
}

type fancyLogger struct {
	w io.Writer
}
