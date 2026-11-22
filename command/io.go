package command

import (
	"errors"
	"io"
	"os"
	"strings"

	flag "github.com/spf13/pflag"
)

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

// OpenInput opens path for reading. `-` means stdin, which is not closed
// when the returned reader is closed.
func OpenInput(path string) (io.ReadCloser, error) {
	if path == "-" {
		return io.NopCloser(os.Stdin), nil
	}
	return os.Open(path)
}

// OpenOutput creates or truncates path for writing. `-` means stdout,
// which is not closed when the returned writer is closed.
func OpenOutput(path string) (io.WriteCloser, error) {
	if path == "-" {
		return nopWriteCloser{os.Stdout}, nil
	}
	return os.Create(path)
}

// ReadSecret reads a secret, such as a password or token, from a file or
// from stdin when path is `-`. A single trailing newline is removed.
//
// Secrets should not be passed as flag values or environment variables,
// since those leak into process listings, shell history and child
// processes. Accept a flag such as --password-file instead, and read it
// with ReadSecret.
func ReadSecret(path string) (string, error) {
	if path == "" {
		return "", errors.New("no secret file given")
	}

	r, err := OpenInput(path)
	if err != nil {
		return "", err
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	secret := strings.TrimSuffix(string(data), "\n")
	return strings.TrimSuffix(secret, "\r"), nil
}

// OptionalStringValue is a flag value whose value may be explicitly turned
// off with the word `none`, such as `--config none`. Empty values are
// rejected, since they are ambiguous.
type OptionalStringValue struct {
	value string
	set   bool
	none  bool
}

// OptionalString defines a string flag that accepts `none` to turn the
// value off, and returns its value.
func OptionalString(f *flag.FlagSet, name string, def string, usage string) *OptionalStringValue {
	v := &OptionalStringValue{value: def}
	f.Var(v, name, usage+`; pass "none" to disable`)
	return v
}

// Set implements pflag.Value.
func (v *OptionalStringValue) Set(s string) error {
	if s == "" {
		return errors.New(`empty value; pass "none" to disable it`)
	}

	v.set = true
	if s == "none" {
		v.none = true
		v.value = ""
		return nil
	}

	v.none = false
	v.value = s
	return nil
}

// String implements pflag.Value.
func (v *OptionalStringValue) String() string {
	if v.none {
		return "none"
	}
	return v.value
}

// Type implements pflag.Value.
func (v *OptionalStringValue) Type() string {
	return "string"
}

// Value returns the flag value, or an empty string when it is `none`.
func (v *OptionalStringValue) Value() string {
	return v.value
}

// IsNone reports whether the flag was set to `none`.
func (v *OptionalStringValue) IsNone() bool {
	return v.none
}

// IsSet reports whether the flag was passed.
func (v *OptionalStringValue) IsSet() bool {
	return v.set
}

var _ flag.Value = (*OptionalStringValue)(nil)
