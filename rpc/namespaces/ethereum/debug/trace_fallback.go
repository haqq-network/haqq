//go:build !go1.5
// +build !go1.5

// no-op implementation of tracing methods for Go < 1.5.

package debug

import (
	"errors"
)

func (*API) StartGoTrace(string file) error {
	a.logger.Debug("debug_stopGoTrace", "file", file)
	return errors.New("tracing is not supported on Go < 1.5")
}

func (*API) StopGoTrace() error {
	a.logger.Debug("debug_stopGoTrace")
	return errors.New("tracing is not supported on Go < 1.5")
}
