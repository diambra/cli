package log

import (
	"bytes"
	"errors"
	"testing"

	"github.com/fatih/color"
)

func colorPrint(at color.Attribute, s string) string {
	return color.New(at).Sprint(FancyPrefix + s + "\n")
}

func TestFancyLogger(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	logger := NewFancyLogger(buf)

	for _, tc := range []struct {
		name string
		kvs  []interface{}
		want string
	}{
		{
			name: "simple",
			kvs:  []interface{}{"msg", "hello world"},
			want: colorPrint(color.Reset, "hello world"),
		},
		{
			name: "error",
			kvs:  []interface{}{"msg", "hello world", "err", errors.New("something went wrong")},
			want: colorPrint(color.Reset, "hello world: something went wrong"),
		},
		{
			name: "id",
			kvs:  []interface{}{"msg", "hello world", "id", "1234567890"},
			want: colorPrint(color.Reset, "(1234) hello world"),
		},
		{
			name: "short id",
			kvs:  []interface{}{"msg", "hello world", "id", "123"},
			want: colorPrint(color.Reset, "(123) hello world"),
		},
		{
			name: "id as int",
			kvs:  []interface{}{"msg", "hello world", "id", 123},
			want: colorPrint(color.Reset, "(123) hello world"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			buf.Reset()
			want := tc.want
			logger.Log(tc.kvs...)
			if got := buf.String(); got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}

}
