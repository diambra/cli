//go:build !windows

package container

import (
	"fmt"
	"os"
	"syscall"
)

func getGID(path string) (uint32, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("couldn't stat /dev/snd: %w", err)
	}
	return fi.Sys().(*syscall.Stat_t).Gid, nil
}
