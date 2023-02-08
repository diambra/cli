package diambra

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/diambra/cli/pkg/container"
)

func configureRender(config *EnvConfig, c *container.Container) error {
	switch {
	case os.Getenv("WAYLAND_DISPLAY") != "":
		return configureWayland(config, c)
	case os.Getenv("DISPLAY") != "":
		return configureX11(config, c)
	default:
		return fmt.Errorf("neither $DISPLAY nor $WAYLAND_DISPLAY are set")
	}
}

func configureX11(config *EnvConfig, c *container.Container) error {
	xauthority := filepath.Join(config.Home, ".Xauthority")
	if xap := os.Getenv("XAUTHORITY"); xap != "" {
		xauthority = xap
	}
	c.BindMounts = append(c.BindMounts,
		container.NewBindMount("/tmp/.X11-unix", "/tmp/.X11-unix"),
		container.NewBindMount(xauthority, "/tmp/.Xauthority"),
	)
	c.Hostname = config.Hostname
	c.Env = append(c.Env, "DISPLAY="+os.Getenv("DISPLAY"))
	c.IPCMode = "host" // We need to enable IPC to avoid "X Error:  BadShmSeg"
	return nil
}

func configureWayland(config *EnvConfig, c *container.Container) error {
	waylandSocket := filepath.Join(os.Getenv("XDG_RUNTIME_DIR"), os.Getenv("WAYLAND_DISPLAY"))
	c.BindMounts = append(c.BindMounts,
		container.NewBindMount(waylandSocket, waylandSocket),
		// container.NewBindMount("/usr/share/X11", "/usr/share/X11"),
	)
	c.Hostname = config.Hostname
	c.Env = append(c.Env, "WAYLAND_DISPLAY="+waylandSocket)
	return nil
}
