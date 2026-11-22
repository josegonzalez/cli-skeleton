package command

import (
	"errors"
	"fmt"
	"strings"
)

// ErrHelp is returned by Parse when help was requested with -h or --help.
var ErrHelp = errors.New("help requested")

// UserError is an expected error, explained in terms the user can act on.
// Execute prints the message, then the hint on its own line, so the
// suggested fix is the last thing on screen.
type UserError struct {
	// Message describes what went wrong.
	Message string

	// Hint suggests how to fix it, such as a command to run.
	Hint string

	// Code is the exit code. Zero means ExitCodeError.
	Code int

	// Err is the underlying error. It is only shown under --debug.
	Err error
}

// NewUserError returns a UserError with a message and an optional hint.
func NewUserError(message string, hint string) *UserError {
	return &UserError{Message: message, Hint: hint}
}

func (e *UserError) Error() string {
	if e.Hint == "" {
		return e.Message
	}
	return e.Message + "\n" + e.Hint
}

func (e *UserError) Unwrap() error {
	return e.Err
}

// exitCode returns the exit code for the error.
func (e *UserError) exitCode() int {
	if e.Code == 0 {
		return ExitCodeError
	}
	return e.Code
}

// UsageError is returned when a command is called with invalid flags or
// arguments. Execute prints it with a pointer to the command help and exits
// with ExitCodeUsage.
type UsageError struct {
	// Err is the parsing or validation error.
	Err error

	// Hint replaces the default pointer to the command help.
	Hint string

	// showUsage prints the concise command help instead of the hint.
	showUsage bool
}

// NewUsageError returns a UsageError with a formatted message.
func NewUsageError(format string, args ...interface{}) *UsageError {
	return &UsageError{Err: fmt.Errorf(format, args...)}
}

func (e *UsageError) Error() string {
	return e.Err.Error()
}

func (e *UsageError) Unwrap() error {
	return e.Err
}

// MultiError groups errors of the same kind under a single header, so they
// are reported together instead of as many similar lines.
type MultiError struct {
	// Header explains what the errors have in common.
	Header string

	// Errors are the grouped errors.
	Errors []error
}

// NewMultiError returns an empty MultiError with the given header.
func NewMultiError(header string) *MultiError {
	return &MultiError{Header: header}
}

// Append adds err to the group. Nil errors are ignored.
func (e *MultiError) Append(err error) {
	if err != nil {
		e.Errors = append(e.Errors, err)
	}
}

// ErrorOrNil returns e when it holds errors, and nil otherwise.
func (e *MultiError) ErrorOrNil() error {
	if e == nil || len(e.Errors) == 0 {
		return nil
	}
	return e
}

func (e *MultiError) Error() string {
	lines := []string{e.Header}
	for _, err := range e.Errors {
		lines = append(lines, "  "+err.Error())
	}
	return strings.Join(lines, "\n")
}

func (e *MultiError) Unwrap() []error {
	return e.Errors
}

// InternalError marks an unexpected error, such as a bug. Execute prints a
// short message and, when Main was given an issue URL, a link to report
// it. The underlying error is only shown under --debug.
type InternalError struct {
	// Err is the unexpected error.
	Err error

	// Stack is the stack trace, when the error came from a panic.
	Stack string
}

// Internal marks err as unexpected.
func Internal(err error) error {
	if err == nil {
		return nil
	}
	return &InternalError{Err: err}
}

func (e *InternalError) Error() string {
	return "unexpected error: " + e.Err.Error()
}

func (e *InternalError) Unwrap() error {
	return e.Err
}

// ExitStatus exits with Code without printing anything, for commands that
// have already reported the problem themselves.
type ExitStatus struct {
	Code int
}

func (e *ExitStatus) Error() string {
	return fmt.Sprintf("exit status %d", e.Code)
}
