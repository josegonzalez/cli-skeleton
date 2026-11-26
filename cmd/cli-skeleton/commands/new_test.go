package commands

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/josegonzalez/cli-skeleton/internal/scaffold"
	"github.com/mitchellh/cli"
)

// scriptedAsker answers questions from a list, and records errors.
type scriptedAsker struct {
	answers   []string
	questions []string
	errors    []string
}

func (s *scriptedAsker) Ask(query string) (string, error) {
	s.questions = append(s.questions, query)
	if len(s.answers) == 0 {
		return "", io.EOF
	}
	answer := s.answers[0]
	s.answers = s.answers[1:]
	return answer, nil
}

func (s *scriptedAsker) Error(message string) {
	s.errors = append(s.errors, message)
}

func TestPromptSpec(t *testing.T) {
	asker := &scriptedAsker{answers: []string{
		"",                    // app name, defaults to the directory
		"example.com/my-app",  // module
		"Does things.",        // description
		"bogus",               // log output, rejected
		"zerolog",             // log output
		"y",                   // config
		"",                    // no command yet, rejected
		"app create",          // command name
		"",                    // synopsis, rejected
		"Create an app",       // synopsis
		"",                    // description
		"force, dry-run",      // standard flags
		"name",                // argument name
		"the name of the app", // argument description
		"",                    // argument type
		"n",                   // optional
		"",                    // no more arguments
		"count",               // flag name
		"int",                 // flag type
		"two",                 // default, rejected with the flag
		"how many",            // flag description
		"count",               // flag name
		"int",                 // flag type
		"2",                   // default
		"how many",            // flag description
		"",                    // no more flags
		"Create an app",       // example description
		"my-app",              // example arguments
		"",                    // no more examples
		"",                    // no more commands
	}}

	spec, err := PromptSpec(asker, "my-app")
	if err != nil {
		t.Fatalf("PromptSpec: %v\nerrors: %v", err, asker.errors)
	}

	want := &scaffold.Spec{
		Name:        "my-app",
		Module:      "example.com/my-app",
		Description: "Does things.",
		UI:          "zerolog",
		Config:      true,
		Commands: []scaffold.Command{{
			Name:      "app create",
			Synopsis:  "Create an app",
			FlagSets:  []string{"client", "force", "dry-run"},
			Arguments: []scaffold.Argument{{Name: "name", Description: "the name of the app"}},
			Flags:     []scaffold.Flag{{Name: "count", Type: "int", Default: "2", Usage: "how many"}},
			Examples:  []scaffold.Example{{Description: "Create an app", Args: "my-app"}},
		}},
	}
	if !reflect.DeepEqual(spec, want) {
		t.Errorf("spec = %#v\nwant %#v", spec, want)
	}
	if len(asker.answers) != 0 {
		t.Errorf("unused answers: %v", asker.answers)
	}

	wantErrors := []string{"Pick one of", "Add at least one command", "An answer is required", `default "two" is not an int`}
	if len(asker.errors) != len(wantErrors) {
		t.Fatalf("errors = %q", asker.errors)
	}
	for i, want := range wantErrors {
		if !strings.Contains(asker.errors[i], want) {
			t.Errorf("errors[%d] = %q, want it to contain %q", i, asker.errors[i], want)
		}
	}
}

func TestPromptSpecRejectsInvalidNames(t *testing.T) {
	a := &scriptedAsker{answers: []string{"My App", "my-app", "", "", "", "", "version", "run", "Run it", "", "", "", "", "", ""}}
	spec, err := PromptSpec(a, "")
	if err != nil {
		t.Fatalf("PromptSpec: %v\nerrors: %v", err, a.errors)
	}
	if spec.Name != "my-app" || len(spec.Commands) != 1 || spec.Commands[0].Name != "run" {
		t.Errorf("spec = %#v", spec)
	}
	if len(a.errors) != 2 || !strings.Contains(a.errors[0], "lowercase") || !strings.Contains(a.errors[1], "added to every app") {
		t.Errorf("errors = %q", a.errors)
	}
}

func TestPromptSpecStopsAtEOF(t *testing.T) {
	if _, err := PromptSpec(&scriptedAsker{}, "app"); !errors.Is(err, io.EOF) {
		t.Errorf("err = %v, want EOF", err)
	}
}

// newCommand returns a NewCommand writing to a mock UI.
func newCommand(t *testing.T) (*NewCommand, *cli.MockUi) {
	t.Helper()
	meta := command.NewMeta(context.Background(), "cli-skeleton", "")
	ui := cli.NewMockUi()
	meta.Ui = ui
	return &NewCommand{Meta: *meta}, ui
}

// messages returns everything written to the mock UI. MockUi writes info
// messages to its OutputWriter.
func messages(ui *cli.MockUi) string {
	return ui.OutputWriter.String() + ui.ErrorWriter.String()
}

// writeSpec writes a spec file and returns its path.
func writeSpec(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "spec.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const runSpec = "name: app\ncommands:\n  - name: run\n    synopsis: Run it\n"

func TestNewWritesTheProject(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "app")
	c, ui := newCommand(t)

	code := c.Run([]string{dir, "--spec", writeSpec(t, runSpec), "--no-tidy"})
	if code != 0 {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, ui.OutputWriter, ui.ErrorWriter)
	}
	for _, path := range []string{"go.mod", "main.go", "main_test.go", "commands/run.go", ".gitignore"} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Errorf("%s was not written: %v", path, err)
		}
	}
	if !strings.Contains(messages(ui), "go get "+scaffold.SkeletonModule) {
		t.Errorf("the skipped go commands are not suggested:\n%s", messages(ui))
	}

	c, ui = newCommand(t)
	code = c.Run([]string{dir, "--spec", writeSpec(t, runSpec), "--no-tidy"})
	if code != command.ExitCodeError {
		t.Fatalf("exit %d, want %d", code, command.ExitCodeError)
	}
	if !strings.Contains(messages(ui), "already exist") || !strings.Contains(messages(ui), "--force") {
		t.Errorf("output = %s", messages(ui))
	}

	c, ui = newCommand(t)
	if code := c.Run([]string{dir, "--spec", writeSpec(t, runSpec), "--no-tidy", "--force"}); code != 0 {
		t.Fatalf("exit %d with --force\nstderr: %s", code, ui.ErrorWriter)
	}
}

func TestNewDryRunWritesNothing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "app")
	c, ui := newCommand(t)

	if code := c.Run([]string{dir, "--spec", writeSpec(t, runSpec), "--dry-run"}); code != 0 {
		t.Fatalf("exit %d\nstderr: %s", code, ui.ErrorWriter)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("%s was created", dir)
	}
	out := ui.OutputWriter.String()
	for _, want := range []string{"Would write " + filepath.Join(dir, "main.go"), "Would run 'go mod tidy'"} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout does not contain %q:\n%s", want, out)
		}
	}
}

func TestNewUsageErrors(t *testing.T) {
	tests := map[string][]string{
		"spec and save-spec": {"--spec", "a.yaml", "--save-spec", "b.yaml"},
		"invalid spec":       {"--spec", writeSpec(t, "name: app\ncommands: []\n")},
		"unknown spec field": {"--spec", writeSpec(t, runSpec+"bogus: true\n")},
		"no terminal":        {"--no-input"},
	}
	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			c, ui := newCommand(t)
			if code := c.Run(append([]string{t.TempDir()}, args...)); code != command.ExitCodeUsage {
				t.Errorf("exit %d, want %d\nstderr: %s", code, command.ExitCodeUsage, ui.ErrorWriter)
			}
		})
	}
}

func TestNewSkeletonPath(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "app")
	c, ui := newCommand(t)

	if code := c.Run([]string{dir, "--spec", writeSpec(t, runSpec), "--skeleton-path", t.TempDir(), "--no-tidy"}); code != command.ExitCodeError {
		t.Fatalf("exit %d for a directory without go.mod\nstderr: %s", code, ui.ErrorWriter)
	}

	skeleton, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	c, ui = newCommand(t)
	if code := c.Run([]string{dir, "--spec", writeSpec(t, runSpec), "--skeleton-path", skeleton, "--no-tidy"}); code != 0 {
		t.Fatalf("exit %d\nstderr: %s", code, ui.ErrorWriter)
	}
	gomod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(gomod), "replace "+scaffold.SkeletonModule+" => ") {
		t.Errorf("go.mod has no replace directive:\n%s", gomod)
	}
	if !strings.Contains(messages(ui), "go mod tidy") || strings.Contains(messages(ui), "go get") {
		t.Errorf("only go mod tidy should be suggested with --skeleton-path:\n%s", messages(ui))
	}
}

func TestReleaseOrLatest(t *testing.T) {
	tests := map[string]string{
		"v1.2.3":                                "v1.2.3",
		"v1.2.3-rc.1":                           "v1.2.3-rc.1",
		"(devel)":                               "latest",
		"":                                      "latest",
		"v0.26.1-0.20261126120000-8d64005abc12": "latest",
		"v0.26.1-0.20261126120000-8d64005abc12+dirty": "latest",
	}
	for version, want := range tests {
		if got := releaseOrLatest(version); got != want {
			t.Errorf("releaseOrLatest(%q) = %q, want %q", version, got, want)
		}
	}
}
