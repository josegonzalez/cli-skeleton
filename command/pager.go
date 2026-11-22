package command

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
)

// PagerCommand returns the pager to use: <APP>_PAGER, then PAGER, falling
// back to less. Setting either variable to an empty string disables
// paging, and so does `cat`.
func PagerCommand(appName string) string {
	if name := AppEnv(appName, "PAGER"); name != "" {
		if value, ok := os.LookupEnv(name); ok {
			return strings.TrimSpace(value)
		}
	}
	if value, ok := os.LookupEnv("PAGER"); ok {
		return strings.TrimSpace(value)
	}
	return "less"
}

// canPage reports whether output should go through a pager: stdin and
// stdout are terminals, --no-pager was not passed, and a pager is set.
func (m *Meta) canPage() bool {
	if m.IsNoPager() || !m.IsInteractive() {
		return false
	}
	pager := PagerCommand(m.appName)
	return pager != "" && pager != "cat"
}

// OutputPaged writes long output to stdout through a pager, such as
// `git diff` does. Output is only paged when stdin and stdout are
// terminals and --no-pager was not passed; otherwise it is written
// directly.
//
// less is run with LESS=FIRX unless LESS is already set: it exits when the
// output fits on one screen, searches case-insensitively, keeps colors and
// leaves the output on screen after exiting.
func (m *Meta) OutputPaged(text string) error {
	if !m.canPage() {
		m.writeOut(text)
		return nil
	}

	args := strings.Fields(PagerCommand(m.appName))
	path, err := exec.LookPath(args[0])
	if err != nil {
		m.writeOut(text)
		return nil
	}

	cmd := exec.Command(path, args[1:]...)
	cmd.Stdin = strings.NewReader(ensureNewline(text))
	cmd.Stdout = m.stdoutWriter()
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if _, ok := os.LookupEnv("LESS"); !ok {
		cmd.Env = append(cmd.Env, "LESS=FIRX")
	}
	if _, ok := os.LookupEnv("LV"); !ok {
		cmd.Env = append(cmd.Env, "LV=-c")
	}

	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return nil
		}
		m.writeOut(text)
	}
	return nil
}

// stdoutWriter returns the stdout of the run.
func (m *Meta) stdoutWriter() io.Writer {
	if m.stdout == nil {
		return os.Stdout
	}
	return m.stdout
}

// writeHelp writes help text to stdout, paging it when it is taller than
// the terminal.
func (m *Meta) writeHelp(text string) {
	height := terminalHeight(m.stdoutFile)
	if height > 0 && strings.Count(text, "\n") >= height && m.canPage() {
		_ = m.OutputPaged(text)
		return
	}
	m.writeOut(text)
}

// helpWriter collects the help mitchellh/cli writes, so it can be paged
// as a whole once the command finishes.
type helpWriter struct {
	meta *Meta
	buf  bytes.Buffer
}

func (w *helpWriter) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

// flush writes the collected help.
func (w *helpWriter) flush() {
	if w.buf.Len() == 0 {
		return
	}
	w.meta.writeHelp(w.buf.String())
	w.buf.Reset()
}
