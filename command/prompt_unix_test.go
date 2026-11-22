//go:build unix

package command

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEditor(t *testing.T) {
	fakeTerminal(t)

	script := filepath.Join(t.TempDir(), "editor.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf 'edited' >> \"$1\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VISUAL", script)
	t.Setenv("TMPDIR", t.TempDir())

	c, _ := newPromptCommand("", nil)
	got, err := c.Editor("original ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "original edited" {
		t.Fatalf("Editor = %q", got)
	}
}
