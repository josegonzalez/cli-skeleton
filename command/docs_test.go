package command

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	flag "github.com/spf13/pflag"
)

func docsCommands() map[string]*testCommand {
	return map[string]*testCommand{
		"eat": {
			name:  "eat",
			flags: FlagSetClient,
			args:  []Argument{{Name: "speed", Type: ArgumentString, Optional: true, Description: "how quickly to eat"}},
			extra: func(f *flag.FlagSet) {
				f.Int("count", 1, "number of lollipops")
			},
			examples: []Example{{Description: "Eat quickly", Command: "test-app eat quickly", Output: "Eating 1 lollipop"}},
		},
		"lollipop list": {name: "lollipop list"},
	}
}

func readDoc(t *testing.T, dir string, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestDocsMarkdown(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "docs")

	got := runTestMain(t, []string{"docs", "--output", dir}, docsCommands(), WithDescription("Eats lollipops."))
	if got.code != 0 {
		t.Fatalf("code = %d, stderr = %q", got.code, got.stderr)
	}

	index := readDoc(t, dir, "index.md")
	for _, want := range []string{"# test-app", "Eats lollipops.", "[`test-app eat`](eat.md): the eat command", "[`test-app lollipop list`](lollipop-list.md)"} {
		if !strings.Contains(index, want) {
			t.Errorf("index missing %q:\n%s", want, index)
		}
	}
	if strings.Contains(index, "docs.md") {
		t.Error("the hidden docs command should not be documented")
	}

	eat := readDoc(t, dir, "eat.md")
	for _, want := range []string{"# test-app eat", "## Usage\n\n```\ntest-app eat [flags] [speed]\n```", "Eat quickly:\n\n```shell\n$ test-app eat quickly\nEating 1 lollipop\n```", "- `[speed]`: how quickly to eat", "--count int", "See [all commands](index.md)."} {
		if !strings.Contains(eat, want) {
			t.Errorf("eat.md missing %q:\n%s", want, eat)
		}
	}
}

func TestDocsMan(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "man")
	t.Setenv("SOURCE_DATE_EPOCH", "1700000000")

	got := runTestMain(t, []string{"docs", "--format", "man", "--output", dir}, docsCommands())
	if got.code != 0 {
		t.Fatalf("code = %d, stderr = %q", got.code, got.stderr)
	}

	index := readDoc(t, dir, "test-app.1")
	if !strings.HasPrefix(index, `.TH "TEST\-APP" "1" "2023-11-14"`) || !strings.Contains(index, `\fBtest\-app\-eat\fR(1)`) {
		t.Errorf("test-app.1:\n%s", index)
	}

	eat := readDoc(t, dir, "test-app-eat.1")
	for _, want := range []string{".SH NAME\ntest\\-app\\-eat \\- the eat command", ".SH SYNOPSIS\n.B test\\-app eat", ".SH OPTIONS", `\fB\-\-count\fR \fIint\fR`, ".SH EXAMPLES", "$ test\\-app eat quickly"} {
		if !strings.Contains(eat, want) {
			t.Errorf("test-app-eat.1 missing %q:\n%s", want, eat)
		}
	}

	if _, err := os.Stat(filepath.Join(dir, "test-app-lollipop-list.1")); err != nil {
		t.Errorf("nested commands should get their own page: %v", err)
	}
}

func TestDocsInstall(t *testing.T) {
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	manPages := filepath.Join(data, "man", "man1")

	got := runTestMain(t, []string{"docs", "--format", "man", "--install"}, docsCommands())
	if got.code != ExitCodeUsage || !strings.Contains(got.stderr, "--force") {
		t.Fatalf("installing without a terminal should need --force: %d %q", got.code, got.stderr)
	}

	got = runTestMain(t, []string{"docs", "--format", "man", "--install", "--force"}, docsCommands())
	if got.code != 0 {
		t.Fatalf("code = %d, stderr = %q", got.code, got.stderr)
	}
	if _, err := os.Stat(filepath.Join(manPages, "test-app-eat.1")); err != nil {
		t.Fatalf("man pages not installed: %v", err)
	}

	got = runTestMain(t, []string{"docs", "--install", "--force"}, docsCommands())
	if got.code != ExitCodeUsage {
		t.Fatalf("--install should require --format man: %d", got.code)
	}
}

func TestDocsUnknownFormat(t *testing.T) {
	got := runTestMain(t, []string{"docs", "--format", "pdf", "--output", t.TempDir()}, docsCommands())
	if got.code != ExitCodeUsage {
		t.Fatalf("code = %d", got.code)
	}
}

func TestDocsCommandIsHidden(t *testing.T) {
	got := runTestMain(t, []string{"--help"}, docsCommands())
	if strings.Contains(got.stdout, "  docs ") {
		t.Fatalf("the docs command should be hidden:\n%s", got.stdout)
	}
}

func TestRoff(t *testing.T) {
	if got := roff(`a-b \ c`); got != `a\-b \e c` {
		t.Errorf("roff = %q", got)
	}
	if got := roff(".starts with a dot\n'quote"); got != "\\&.starts with a dot\n\\&'quote" {
		t.Errorf("roff = %q", got)
	}
}
