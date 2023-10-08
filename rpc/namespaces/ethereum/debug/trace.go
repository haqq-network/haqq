//go:build go1.5
// +build go1.5

package debug

import (
	"errors"
	"os"
	"runtime/trace"

	stderrors "github.com/pkg/errors"
)

// StartGoTrace turns on tracing, writing to the given file.
func (a *API) StartGoTrace(file string) error {
	a.logger.Debug("debug_startGoTrace", "file", file)
	a.handler.mu.Lock()
	defer a.handler.mu.Unlock()

	if a.handler.traceFile != nil {
		a.logger.Debug("trace already in progress")
		return errors.New("trace already in progress")
	}
	fp, err := ExpandHome(file)
	if err != nil {
		a.logger.Debug("failed to get filepath for the CPU profile file", "error", err.Error())
		return err
	}
	f, err := os.Create(fp)
	if err != nil {
		a.logger.Debug("failed to create go trace file", "error", err.Error())
		return err
	}
	if err := trace.Start(f); err != nil {
		a.logger.Debug("Go tracing already started", "error", err.Error())
		if err := f.Close(); err != nil {
			a.logger.Debug("failed to close trace file")
			return stderrors.Wrap(err, "failed to close trace file")
		}

		return err
	}
	a.handler.traceFile = f
	a.handler.traceFilename = file
	a.logger.Info("Go tracing started", "dump", a.handler.traceFilename)
	return nil
}

// StopGoTrace stops an ongoing trace.
func (a *API) StopGoTrace() error {
	a.logger.Debug("debug_stopGoTrace")
	a.handler.mu.Lock()
	defer a.handler.mu.Unlock()

	trace.Stop()
	if a.handler.traceFile == nil {
		a.logger.Debug("trace not in progress")
		return errors.New("trace not in progress")
	}
	a.logger.Info("Done writing Go trace", "dump", a.handler.traceFilename)
	if err := a.handler.traceFile.Close(); err != nil {
		a.logger.Debug("failed to close trace file")
		return stderrors.Wrap(err, "failed to close trace file")
	}
	a.handler.traceFile = nil
	a.handler.traceFilename = ""
	return nil
}
