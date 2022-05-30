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
	wcErr     chan error
	rcErr     chan error
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
		wcErr:   make(chan error),
		rcErr:   make(chan error),
		closeCh: make(chan struct{}),
	}
}
func (s *streamer) MakeRaw() (err error) {
	// Do not make term raw if we don't read from stdin anyway
	if s.wc == nil {
		return nil
	}
	level.Debug(s.Logger).Log("msg", "Setting stdout to raw")
	s.termState, err = term.MakeRaw(int(os.Stdout.Fd()))
	return err
}
func (s *streamer) Restore() error {
	if s.termState == nil {
		return nil
	}
	return term.Restore(int(os.Stdout.Fd()), s.termState)
}

func (s *streamer) Stream() error {
	var ()
	level.Debug(s.Logger).Log("msg", "starting stream")
	if err := s.MakeRaw(); err != nil {
		return err
	}
	go func() {
		<-s.closeCh
		if s.wc != nil {
			s.wc.Close()
		}
		s.rc.Close()
		level.Debug(s.Logger).Log("msg", "closed wc and rc")
	}()

	go func() {
		_, err := io.Copy(os.Stdout, s.rc)
		s.Restore()
		if !s.done && err != nil {
			level.Warn(s.Logger).Log("msg", "copy output failed", "err", err.Error())
		}
		level.Debug(s.Logger).Log("msg", "copying to stdout done, signaling")
		s.rcErr <- err
		level.Debug(s.Logger).Log("msg", "signaled")
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
		level.Debug(s.Logger).Log("msg", "copying to stdin done, signaling")
		s.wcErr <- err
		level.Debug(s.Logger).Log("msg", "signaled")
	}()
	level.Debug(s.Logger).Log("msg", "returning stream")
	return nil
}

func (s *streamer) Close() {
	s.done = true // set done to ignore close errors. FIXME: Is there a better way?
	s.closeCh <- struct{}{}

	var (
		wcDone = false
		rcDone = false
	)
	for {
		if wcDone && rcDone {
			return
		}
		select {
		case <-s.wcErr:
			rcDone = true
		case <-s.rcErr:
			wcDone = true
		}
	}
}
