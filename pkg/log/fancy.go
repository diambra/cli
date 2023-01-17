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

const FancyPrefix = "\033[0G\033[2K"

func NewFancyLogger(w io.Writer) log.Logger {
	return &fancyLogger{w}
}

func format(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case error:
		return val.Error()
	case int:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%v", v)
	}
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
				errstr = ": " + format(val)
			case "msg":
				msg = format(val)
			case "id":
				fullId := format(val)
				// Use up to first 4 characters of the ID
				l := len(fullId)
				if l > 4 {
					l = 4
				}
				id = fmt.Sprintf("(%s) ", fullId[:l])
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
				c, ok := Colors[val.(level.Value).String()]
				if ok {
					col = c
				}
			}
		}
	}
	col.Fprintf(l.w, "%s%s%s%s%s\n", FancyPrefix, icon, id, msg, errstr)
	return nil
}

var Colors = map[string]*color.Color{
	"warn":  color.New(color.FgYellow),
	"error": color.New(color.FgRed),
	"info":  color.New(color.FgHiWhite),
	"debug": color.New(color.FgHiBlack),
}

type fancyLogger struct {
	w io.Writer
}
