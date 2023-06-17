package errs

import (
	"fmt"
	"go.uber.org/multierr"
	"testing"
)

// Capture runs errF and assigns the error, if any, to *err. Preserves the
// original err by wrapping in a MultiError if err is non-nil. If msg is not
// empty, wrap the error returned by closer with the msg.
//
//   - If errF returns nil, do nothing.
//   - If errF returns an error and *err == nil, replace *err with the error.
//   - If errF returns an error and *err != nil, replace *err with a MultiError
//     containing *err and the errF err.
func Capture(err *error, errF func() error, msg string) {
	fErr := errF()
	if fErr == nil {
		return
	}

	wErr := fErr
	if msg != "" {
		wErr = fmt.Errorf(msg+": %w", fErr)
	}
	if *err == nil {
		// Only 1 error so replace the err pointer.
		*err = wErr
		return
	}

	multierr.AppendInto(err, wErr)
}

// CaptureT call t.Error if errF returns an error with an optional message.
func CaptureT(t *testing.T, errF func() error, msg string) {
	t.Helper()
	if err := errF(); err != nil {
		if msg == "" {
			t.Error(err)
		} else {
			t.Errorf(msg+": %s", err)
		}
	}
}
