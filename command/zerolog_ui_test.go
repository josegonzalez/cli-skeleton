package command

import (
	"bytes"
	"strings"
	"testing"
)

func newTestZerologUi(human bool, color bool) (*ZerologUi, *bytes.Buffer, *bytes.Buffer) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ui, _, _ := newTestTerminalUi("", color)
	return newZerologUi(ui, map[string]interface{}{}, zerologUiOptions{
		human:       human,
		stdout:      stdout,
		stderr:      stderr,
		stdoutColor: color,
		stderrColor: color,
	}), stdout, stderr
}

func TestHumanZerologUiWithoutColor(t *testing.T) {
	ui, stdout, stderr := newTestZerologUi(true, false)

	ui.LogHeader1("header")
	ui.Output("output")
	ui.Error("error")
	ui.Warn("warn")

	all := stdout.String() + stderr.String()
	if strings.Contains(all, "\x1b[") {
		t.Fatalf("escape codes written with color disabled: %q", all)
	}
	if !strings.Contains(stderr.String(), "=====> header") {
		t.Errorf("header missing: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), " !     error") {
		t.Errorf("error missing: %q", stderr.String())
	}
}

func TestHumanZerologUiWithColor(t *testing.T) {
	ui, _, stderr := newTestZerologUi(true, true)

	ui.Error("error")
	if !strings.Contains(stderr.String(), "\x1b[") {
		t.Fatalf("expected escape codes with color enabled: %q", stderr.String())
	}
}

func TestZerologUiWithoutColor(t *testing.T) {
	ui, stdout, stderr := newTestZerologUi(false, false)

	ui.Info("info")
	ui.Error("error")

	all := stdout.String() + stderr.String()
	if strings.Contains(all, "\x1b[") {
		t.Fatalf("escape codes written with color disabled: %q", all)
	}
}

func TestZerologUiFieldKeepsFormatAndWriters(t *testing.T) {
	ui, stdout, _ := newTestZerologUi(true, false)

	child := ui.Field("app", "lollipop")
	child.Output("output")

	if !child.human || !child.OutputIndentField {
		t.Fatal("Field() should keep the human-readable format")
	}
	if !strings.Contains(stdout.String(), "output") || !strings.Contains(stdout.String(), "app=lollipop") {
		t.Fatalf("child output not written to the parent writer: %q", stdout.String())
	}
	if _, ok := ui.OriginalFields["app"]; ok {
		t.Fatal("Field() should not modify the parent fields")
	}
}

func TestZerologUiFollowsWrappedUiColors(t *testing.T) {
	inner, _, _ := newTestTerminalUi("", false)
	inner.ErrorColor = true

	ui := HumanZerologUiWithFields(inner, nil)
	stdoutColor, stderrColor := ui.ColorEnabled()
	if stdoutColor || !stderrColor {
		t.Fatalf("ColorEnabled() = %v, %v", stdoutColor, stderrColor)
	}
}

func TestZerologUiStreams(t *testing.T) {
	for _, human := range []bool{true, false} {
		ui, stdout, stderr := newTestZerologUi(human, false)

		ui.Output("the-output")
		ui.Info("the-info")
		ui.LogHeader1("the-header1")
		ui.LogHeader2("the-header2")
		ui.Warn("the-warn")
		ui.Error("the-error")

		if !strings.Contains(stdout.String(), "the-output") {
			t.Errorf("human=%v: output missing from stdout: %q", human, stdout.String())
		}
		for _, message := range []string{"the-info", "the-header1", "the-header2", "the-warn", "the-error"} {
			if strings.Contains(stdout.String(), message) {
				t.Errorf("human=%v: %s written to stdout: %q", human, message, stdout.String())
			}
			if !strings.Contains(stderr.String(), message) {
				t.Errorf("human=%v: %s missing from stderr: %q", human, message, stderr.String())
			}
		}
	}
}
