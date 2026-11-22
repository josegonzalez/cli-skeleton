package command

import (
	"bufio"
	"bytes"
	"fmt"

	"github.com/mitchellh/cli"
	"github.com/posener/complete"
)

// MergeAutocompleteFlags is used to join multiple flag completion sets.
func MergeAutocompleteFlags(flags ...complete.Flags) complete.Flags {
	merged := make(map[string]complete.Predictor, len(flags))
	for _, f := range flags {
		for k, v := range f {
			merged[k] = v
		}
	}
	return merged
}

// CommandErrorText is used to easily render the same messaging across commads
// when an error is printed.
func CommandErrorText(cmd NamedCommand) string {
	return fmt.Sprintf("For additional help try '%s %s --help'", cmd.AppName(), cmd.Name())
}

// uiErrorWriter is a io.Writer that writes lines as warnings through the
// UI. Meta.FlagSet uses it for the flag set output, which only receives
// notices such as flag deprecation warnings, since parse errors are
// returned and reported by Execute.
//
// uiErrorWriter scans input for individual lines to pass to the UI. If data
// doesn't contain a new line, it buffers result until next new line or writer is closed.
type uiErrorWriter struct {
	ui  cli.Ui
	buf bytes.Buffer
}

func (w *uiErrorWriter) Write(data []byte) (int, error) {
	read := 0
	for len(data) != 0 {
		a, token, err := bufio.ScanLines(data, false)
		if err != nil {
			return read, err
		}

		if a == 0 {
			r, err := w.buf.Write(data)
			return read + r, err
		}

		w.ui.Warn(w.buf.String() + string(token))
		data = data[a:]
		w.buf.Reset()
		read += a
	}

	return read, nil
}

func (w *uiErrorWriter) Close() error {
	// emit what's remaining
	if w.buf.Len() != 0 {
		w.ui.Warn(w.buf.String())
		w.buf.Reset()
	}
	return nil
}
