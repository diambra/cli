package diambra

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppArgs(t *testing.T) {
	for _, tc := range []struct {
		name     string
		appArgs  AppArgs
		expected []string
	}{
		{
			"empty",
			AppArgs{
				RandomSeed: 0,
				Render:     false,
				LockFPS:    false,
				Sound:      false,
			},
			[]string{},
		},
		{
			"full",
			AppArgs{
				RandomSeed: 23,
				Render:     true,
				LockFPS:    true,
				Sound:      true,
			},
			[]string{"--render", "--lockFps", "--sound", "--randomSeed", "23"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.appArgs.Args(), tc.expected)
		})
	}
}
