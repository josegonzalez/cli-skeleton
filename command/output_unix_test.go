//go:build unix

package command

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writePager writes a pager script that records its input and environment.
func writePager(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	out := filepath.Join(dir, "paged.txt")
	script := filepath.Join(dir, "pager.sh")
	contents := "#!/bin/sh\necho \"LESS=$LESS\" > \"" + out + "\"\ncat >> \"" + out + "\"\n"
	if err := os.WriteFile(script, []byte(contents), 0o700); err != nil {
		t.Fatal(err)
	}
	return script, out
}

func TestOutputPagedUsesPager(t *testing.T) {
	fakeTerminal(t)
	fakeStdoutTerminal(t)
	script, out := writePager(t)
	t.Setenv("PAGER", script)
	t.Setenv("LESS", "")
	os.Unsetenv("LESS")

	c, stdout, _ := newExecuteCommand(nil)
	if err := c.OutputPaged("long output"); err != nil {
		t.Fatal(err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("paged output should not be written directly: %q", stdout.String())
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "LESS=FIRX\nlong output\n" {
		t.Fatalf("pager received %q", data)
	}
}

func TestOutputPagedKeepsLESS(t *testing.T) {
	fakeTerminal(t)
	fakeStdoutTerminal(t)
	script, out := writePager(t)
	t.Setenv("PAGER", script)
	t.Setenv("LESS", "-S")

	c, _, _ := newExecuteCommand(nil)
	c.OutputPaged("long output")

	data, _ := os.ReadFile(out)
	if !strings.HasPrefix(string(data), "LESS=-S\n") {
		t.Fatalf("pager received %q", data)
	}
}

func TestHelpIsPagedWhenTallerThanTerminal(t *testing.T) {
	fakeTerminal(t)
	fakeStdoutTerminal(t)
	script, out := writePager(t)
	t.Setenv("PAGER", script)
	t.Setenv("LINES", "3")

	c, stdout, _ := newExecuteCommand(nil)
	c.writeHelp("one\ntwo\nthree\nfour\n")
	if stdout.Len() != 0 {
		t.Fatalf("tall help should be paged: %q", stdout.String())
	}
	if data, _ := os.ReadFile(out); !strings.Contains(string(data), "four") {
		t.Fatalf("pager received %q", data)
	}

	c, stdout, _ = newExecuteCommand(nil)
	c.writeHelp("one\n")
	if stdout.String() != "one\n" {
		t.Fatalf("short help should not be paged: %q", stdout.String())
	}
}
