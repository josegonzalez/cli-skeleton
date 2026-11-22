package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"text/tabwriter"
	"unicode"
)

// stdoutIsTerminal reports whether the stdout of the run is a terminal.
// Tests replace it to simulate a terminal.
var stdoutIsTerminal = func(m *Meta) bool {
	f := m.stdoutFile
	if f == nil && m.stdout == nil {
		f = os.Stdout
	}
	return IsTerminal(f)
}

// IsInteractive reports whether both stdin and stdout are terminals, so a
// person is likely reading the output and can answer prompts.
func (m *Meta) IsInteractive() bool {
	return isTerminalReader(m.stdinReader()) && stdoutIsTerminal(m)
}

// IsStdoutTerminal reports whether stdout is a terminal.
func (m *Meta) IsStdoutTerminal() bool {
	return stdoutIsTerminal(m)
}

// IsStderrTerminal reports whether stderr is a terminal. Animations such
// as spinners and progress bars should only be drawn when it is.
func (m *Meta) IsStderrTerminal() bool {
	if f, ok := m.stderr.(*os.File); ok {
		return IsTerminal(f)
	}
	if m.stderr == nil {
		return IsTerminal(os.Stderr)
	}
	return m.stderrFile != nil && IsTerminal(m.stderrFile)
}

// OutputJSON writes v to stdout as indented JSON. JSON is machine-readable
// output, so it bypasses any log formatting of the UI.
func (m *Meta) OutputJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	m.writeOut(string(data))
	return nil
}

// OutputTable writes rows of data in the format the user asked for:
//
//   - by default, aligned columns under a header row;
//   - with --plain, one tab-separated record per line without a header, for
//     tools such as grep, cut and awk;
//   - with --json, an array of objects keyed by the headers in snake_case.
func (m *Meta) OutputTable(headers []string, rows [][]string) error {
	if m.flags.json {
		records := make([]map[string]string, 0, len(rows))
		for _, row := range rows {
			record := make(map[string]string, len(headers))
			for i, header := range headers {
				if i < len(row) {
					record[jsonKey(header)] = row[i]
				} else {
					record[jsonKey(header)] = ""
				}
			}
			records = append(records, record)
		}
		return m.OutputJSON(records)
	}

	if m.flags.plain {
		lines := make([]string, 0, len(rows))
		for _, row := range rows {
			lines = append(lines, strings.Join(cleanCells(row), "\t"))
		}
		if len(lines) > 0 {
			m.writeOut(strings.Join(lines, "\n"))
		}
		return nil
	}

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)
	header := make([]string, len(headers))
	for i, h := range headers {
		header[i] = strings.ToUpper(h)
	}
	fmt.Fprintln(w, strings.Join(header, "\t"))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(cleanCells(row), "\t"))
	}
	if err := w.Flush(); err != nil {
		return err
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	if m.stdoutColor() && len(lines) > 0 {
		lines[0] = "\x1b[1m" + lines[0] + "\x1b[0m"
	}
	m.writeOut(strings.Join(lines, "\n"))
	return nil
}

// cleanCells replaces tabs and newlines in cells with spaces, so each record
// stays on one line and columns stay aligned.
func cleanCells(row []string) []string {
	replacer := strings.NewReplacer("\t", " ", "\r\n", " ", "\n", " ")
	cells := make([]string, len(row))
	for i, cell := range row {
		cells[i] = replacer.Replace(cell)
	}
	return cells
}

// jsonKey turns a table header into a snake_case JSON key.
func jsonKey(header string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(header) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

// Success tells the user what a command changed, such as "Deployed my-app".
// It is written to stdout with a check mark on a color terminal, and
// omitted under --json, where the JSON output is the result.
func (m *Meta) Success(message string) {
	if m.flags.json {
		return
	}
	if m.stdoutColor() {
		message = colorize(successMark(), colorGreen, false) + " " + message
	}
	m.Ui.Output(message)
}

// successMark returns the check mark used by Success.
func successMark() string {
	if supportsUTF8() {
		return "✓"
	}
	return "*"
}

// Suggest tells the user which commands to run next, such as
// `app status`. It is written to stderr, and hidden under --quiet and
// --json.
func (m *Meta) Suggest(commands ...string) {
	if m.flags.json || m.IsQuiet() || len(commands) == 0 {
		return
	}

	if len(commands) == 1 {
		m.Ui.Info(fmt.Sprintf("Next, run '%s'.", commands[0]))
		return
	}

	lines := []string{"Next steps:"}
	for _, command := range commands {
		lines = append(lines, "  "+command)
	}
	m.Ui.Info(strings.Join(lines, "\n"))
}

// supportsUTF8 reports whether the terminal locale is UTF-8.
func supportsUTF8() bool {
	if runtime.GOOS == "windows" {
		return true
	}
	for _, name := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		if value := os.Getenv(name); value != "" {
			value = strings.ToLower(value)
			return strings.Contains(value, "utf-8") || strings.Contains(value, "utf8")
		}
	}
	return false
}
