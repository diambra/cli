package container

import (
	"fmt"
	"io"
	"os"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"golang.org/x/term"
)

type streamer struct {
	log.Logger
	wc        io.WriteCloser
	rc        io.ReadCloser
	termState *term.State
	closeCh   chan struct{}
	done      bool
}

func NewStreamer(logger log.Logger, wc io.WriteCloser, rc io.ReadCloser) *streamer {
	level.Debug(logger).Log("msg", "creating streamer", "wc", fmt.Sprintf("%v", wc), "rc", fmt.Sprintf("%v", rc))
	return &streamer{
		Logger:  logger,
		wc:      wc,
		rc:      rc,
		closeCh: make(chan struct{}),
	}
}
func (s *streamer) MakeRaw() (err error) {
	// Do not make term raw if we don't read from stdin anyway
	if s.wc == nil {
		return nil
	}
	level.Debug(s.Logger).Log("Setting stdout to raw")
	s.termState, err = term.MakeRaw(int(os.Stdout.Fd()))
	return err
}
func (s *streamer) Restore() error {
	if s.termState == nil {
		return nil
	}
	return term.Restore(int(os.Stdout.Fd()), s.termState)
}

func (s *streamer) Stream() (<-chan error, <-chan error, error) {
	var (
		wcErr = make(chan error)
		rcErr = make(chan error)
	)
	level.Debug(s.Logger).Log("msg", "starting stream")
	if err := s.MakeRaw(); err != nil {
		return nil, nil, err
	}
	go func() {
		<-s.closeCh
		s.done = true // set done to ignore close errors. FIXME: Is there a better way?
		if s.wc != nil {
			s.wc.Close()
		}
		s.rc.Close()
	}()

	go func() {
		_, err := io.Copy(os.Stdout, s.rc)
		s.Restore()
		if !s.done && err != nil {
			level.Warn(s.Logger).Log("msg", "copy output failed", "err", err.Error())
		}
		rcErr <- err
	}()

	go func() {
		if s.wc == nil { // No stdin
			return
		}
		_, err := io.Copy(s.wc, os.Stdin)
		s.Restore()
		if !s.done && err != nil {
			level.Warn(s.Logger).Log("msg", "copy input failed", "err", err.Error())
		}
		wcErr <- err
	}()
	level.Debug(s.Logger).Log("msg", "returning stream")
	return wcErr, rcErr, nil
}

func (s *streamer) Close() {
	s.closeCh <- struct{}{}
}
