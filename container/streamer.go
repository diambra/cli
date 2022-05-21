package container

import (
	"io"
	"os"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"golang.org/x/term"
)

type streamer struct {
	log.Logger
	wc      io.WriteCloser
	rc      io.ReadCloser
	closeCh chan struct{}
	done    bool
}

func NewStreamer(logger log.Logger, wc io.WriteCloser, rc io.ReadCloser) *streamer {
	return &streamer{
		Logger:  logger,
		wc:      wc,
		rc:      rc,
		closeCh: make(chan struct{}),
	}
}

func (s *streamer) Stream() error {
	termState, err := term.MakeRaw(int(os.Stdout.Fd()))
	if err != nil {
		return err
	}

	go func() {
		<-s.closeCh
		s.done = true // set done to ignore close errors. FIXME: Is there a better way?
		s.wc.Close()
		s.rc.Close()
	}()

	go func() {
		_, err := io.Copy(os.Stdout, s.rc)
		term.Restore(int(os.Stdout.Fd()), termState)
		if !s.done && err != nil {
			level.Warn(s.Logger).Log("msg", "copy output failed", "err", err.Error())
		}
	}()

	go func() {
		_, err := io.Copy(s.wc, os.Stdin)
		if !s.done && err != nil {
			level.Warn(s.Logger).Log("msg", "copy input failed", "err", err.Error())
		}
	}()
	return nil
}

func (s *streamer) Close() {
	s.closeCh <- struct{}{}
}
