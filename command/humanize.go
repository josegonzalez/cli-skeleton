package command

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"syscall"
)

// HumanizeError rewrites common low-level errors as UserErrors that explain
// the problem and suggest a fix. Other errors are returned unchanged. The
// original error is kept, and shown under --debug.
//
// Execute calls it for every error a command returns, so commands only need
// to call it to inspect the result themselves.
func HumanizeError(err error) error {
	if err == nil {
		return nil
	}

	var userErr *UserError
	var usageErr *UsageError
	var multiErr *MultiError
	var internalErr *InternalError
	if errors.As(err, &userErr) || errors.As(err, &usageErr) || errors.As(err, &multiErr) || errors.As(err, &internalErr) {
		return err
	}

	var pathErr *fs.PathError
	hasPath := errors.As(err, &pathErr)

	switch {
	case errors.Is(err, fs.ErrPermission) && hasPath:
		return &UserError{
			Message: fmt.Sprintf("Permission denied: %s", pathErr.Path),
			Hint:    fmt.Sprintf("Check the permissions with 'ls -l %s', or change them with 'chmod'.", pathErr.Path),
			Err:     err,
		}

	case errors.Is(err, fs.ErrNotExist) && hasPath:
		return &UserError{
			Message: fmt.Sprintf("No such file or directory: %s", pathErr.Path),
			Hint:    "Check that the path is spelled correctly and exists.",
			Err:     err,
		}

	case errors.Is(err, syscall.ECONNREFUSED):
		message := "Connection refused"
		var opErr *net.OpError
		if errors.As(err, &opErr) && opErr.Addr != nil {
			message = fmt.Sprintf("Connection refused by %s", opErr.Addr.String())
		}
		return &UserError{
			Message: message,
			Hint:    "Check that the server is running and reachable.",
			Err:     err,
		}

	case isTimeout(err):
		return &UserError{
			Message: "The operation timed out.",
			Hint:    "Try again, or allow more time with --timeout.",
			Err:     err,
		}
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) && !dnsErr.IsTimeout {
		return &UserError{
			Message: fmt.Sprintf("Unable to resolve %s", dnsErr.Name),
			Hint:    "Check the hostname and your network connection.",
			Err:     err,
		}
	}

	return err
}

// isTimeout reports whether err is a deadline or network timeout.
func isTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}

	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
