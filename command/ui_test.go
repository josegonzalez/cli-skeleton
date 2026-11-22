package command

import (
	"bytes"
	"strings"
	"testing"
)

func newTestTerminalUi(input string, color bool) (*TerminalUi, *bytes.Buffer, *bytes.Buffer) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	return &TerminalUi{
		Reader:      strings.NewReader(input),
		Writer:      stdout,
		ErrorWriter: stderr,
		OutputColor: color,
		ErrorColor:  color,
	}, stdout, stderr
}

func TestTerminalUiWithoutColor(t *testing.T) {
	ui, stdout, stderr := newTestTerminalUi("", false)

	ui.Output("output")
	ui.Error("error")
	ui.Warn("warn")

	if got := stdout.String(); got != "output\n" {
		t.Errorf("stdout = %q", got)
	}
	if got := stderr.String(); got != "error\nwarn\n" {
		t.Errorf("stderr = %q", got)
	}
	if strings.Contains(stdout.String()+stderr.String(), "\x1b[") {
		t.Error("escape codes written with color disabled")
	}
}

func TestTerminalUiColorsErrorsAndWarningsOnly(t *testing.T) {
	ui, stdout, stderr := newTestTerminalUi("", true)

	ui.Output("output")
	ui.Info("info")
	ui.Error("error")
	ui.Warn("warn")

	if strings.Contains(stdout.String(), "\x1b[") {
		t.Errorf("output and info should not be colored: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "\x1b[31merror\x1b[0m") {
		t.Errorf("error should be red: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "\x1b[33mwarn\x1b[0m") {
		t.Errorf("warn should be yellow: %q", stderr.String())
	}
}

func TestTerminalUiColorsStreamsIndependently(t *testing.T) {
	ui, _, stderr := newTestTerminalUi("", false)
	ui.ErrorColor = true
	ui.Error("error")

	if !strings.Contains(stderr.String(), "\x1b[31m") {
		t.Errorf("stderr color should not depend on stdout: %q", stderr.String())
	}

	stdoutColor, stderrColor := ui.ColorEnabled()
	if stdoutColor || !stderrColor {
		t.Errorf("ColorEnabled() = %v, %v", stdoutColor, stderrColor)
	}
}

func TestTerminalUiAsk(t *testing.T) {
	ui, stdout, _ := newTestTerminalUi("first\r\nsecond\nlast", false)

	for _, want := range []string{"first", "second", "last"} {
		got, err := ui.Ask("Name?")
		if err != nil {
			t.Fatalf("Ask: %v", err)
		}
		if got != want {
			t.Errorf("Ask = %q, want %q", got, want)
		}
	}

	if _, err := ui.Ask("Name?"); err == nil {
		t.Error("expected EOF error after input is exhausted")
	}

	if !strings.Contains(stdout.String(), "Name? ") {
		t.Errorf("query not written: %q", stdout.String())
	}
}

func TestTerminalUiAskSecretWithoutTerminal(t *testing.T) {
	ui, _, _ := newTestTerminalUi("hunter2\n", false)

	got, err := ui.AskSecret("Password:")
	if err != nil {
		t.Fatalf("AskSecret: %v", err)
	}
	if got != "hunter2" {
		t.Errorf("AskSecret = %q", got)
	}
}
