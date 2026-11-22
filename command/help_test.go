package command

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
	flag "github.com/spf13/pflag"
)

type describedCommand struct {
	testCommand
}

func (c *describedCommand) Description() string {
	return "Lollipops are eaten one at a time, in the order they were bought."
}

func newHelpCommand() *testCommand {
	c, _, _ := newHelpCommandWithOutput()
	return c
}

func newHelpCommandWithOutput() (*testCommand, *bytes.Buffer, *bytes.Buffer) {
	c, stdout, stderr := newExecuteCommand(nil)
	c.extra = func(f *flag.FlagSet) {
		f.Int("count", 1, "number of lollipops to eat")
	}
	c.examples = []Example{
		{Description: "Eats one lollipop quickly", Command: "test-app eat quickly", Output: "Eating 1 lollipop quickly"},
		{Description: "Eats two lollipops", Command: "test-app eat --count 2"},
	}
	return c, stdout, stderr
}

func TestCommandHelpLayout(t *testing.T) {
	t.Setenv("COLUMNS", "")
	os.Unsetenv("COLUMNS")
	c := newHelpCommand()

	got := CommandHelp(c)

	order := []string{"USAGE", "test-app eat [flags] [speed]", "the eat command", "EXAMPLES", "ARGUMENTS", "OPTIONS"}
	last := -1
	for _, want := range order {
		i := strings.Index(got, want)
		if i < 0 {
			t.Fatalf("help missing %q:\n%s", want, got)
		}
		if i < last {
			t.Fatalf("%q out of order:\n%s", want, got)
		}
		last = i
	}

	for _, want := range []string{
		"  Eats one lollipop quickly\n    $ test-app eat quickly\n    Eating 1 lollipop quickly",
		"  [speed] string",
		"--count int",
		"--quiet",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("help missing %q:\n%s", want, got)
		}
	}

	if strings.Contains(got, "\x1b[") {
		t.Error("help should not contain escape codes without color")
	}
	if strings.Contains(got, "LEARN MORE") {
		t.Error("LEARN MORE should only be shown with docs or support URLs")
	}
}

func TestCommandHelpIsStable(t *testing.T) {
	c := newHelpCommand()
	first := CommandHelp(c)
	for i := 0; i < 20; i++ {
		if got := CommandHelp(c); got != first {
			t.Fatalf("help changed between runs:\n%s\n---\n%s", first, got)
		}
	}
	if strings.Index(first, "Eats one lollipop quickly") > strings.Index(first, "Eats two lollipops") {
		t.Fatal("examples should keep their declared order")
	}
}

func TestCommandHelpWithoutFlags(t *testing.T) {
	c, _, _ := newExecuteCommand(nil)
	c.flags = FlagSetNone
	c.args = nil

	got := CommandHelp(c)
	if !strings.Contains(got, "USAGE\n  test-app eat\n") {
		t.Fatalf("usage should omit [flags] without flags:\n%s", got)
	}
	if strings.Contains(got, "OPTIONS") || strings.Contains(got, "ARGUMENTS") || strings.Contains(got, "EXAMPLES") {
		t.Fatalf("empty sections should be omitted:\n%s", got)
	}
}

func TestCommandHelpBoldHeadings(t *testing.T) {
	c := newHelpCommand()
	inner := c.Ui.(*settingsUi).Ui.(*TerminalUi)
	inner.OutputColor = true

	got := CommandHelp(c)
	if !strings.Contains(got, "\x1b[1mUSAGE\x1b[0m") {
		t.Fatalf("headings should be bold on a color terminal:\n%q", got)
	}
}

func TestCommandHelpDescriptionAndLinks(t *testing.T) {
	inner := newHelpCommand()
	c := &describedCommand{testCommand: *inner}
	c.docsURL = "https://example.com/docs/"
	c.supportURL = "https://example.com/support"
	c.name = "lollipop eat"

	got := CommandHelp(c)
	for _, want := range []string{
		"the lollipop eat command\n\nLollipops are eaten one at a time",
		"LEARN MORE",
		"Read the docs at https://example.com/docs/lollipop-eat",
		"Get help or report issues at https://example.com/support",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("help missing %q:\n%s", want, got)
		}
	}
}

func TestCommandHelpWraps(t *testing.T) {
	t.Setenv("COLUMNS", "40")
	inner := newHelpCommand()
	c := &describedCommand{testCommand: *inner}

	got := CommandHelp(c)
	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(line, "Lollipops") && len(line) > 40 {
			t.Fatalf("description not wrapped to COLUMNS: %q", line)
		}
	}
}

func TestConciseHelpOnMissingArguments(t *testing.T) {
	c, _, stderr := newHelpCommandWithOutput()
	c.args = []Argument{{Name: "flavor", Type: ArgumentString}}

	code := Execute(c, nil, func(map[string]Argument) error {
		t.Fatal("run should not be called")
		return nil
	})
	if code != ExitCodeUsage {
		t.Fatalf("exit code = %d", code)
	}

	out := stderr.String()
	for _, want := range []string{"requires 1 argument", "USAGE", "test-app eat [flags] <flavor>", "Eats one lollipop quickly", "Run 'test-app eat --help' for more information."} {
		if !strings.Contains(out, want) {
			t.Errorf("concise help missing %q:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"Eats two lollipops", "OPTIONS"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("concise help should not contain %q:\n%s", unwanted, out)
		}
	}
}

func TestUnknownFlagSuggestion(t *testing.T) {
	c, _, stderr := newHelpCommandWithOutput()

	code := Execute(c, []string{"--cuont", "2"}, func(map[string]Argument) error { return nil })
	if code != ExitCodeUsage {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stderr.String(), "Did you mean --count?") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestMainNoArgsShowsConciseHelp(t *testing.T) {
	commands := map[string]*testCommand{"eat": {name: "eat"}}

	got := runTestMain(t, nil, commands,
		WithDescription("Eats lollipops."),
		WithExamples(
			Example{Description: "first", Command: "test-app eat"},
			Example{Description: "second", Command: "test-app eat fast"},
			Example{Description: "third", Command: "test-app eat slow"},
		),
	)
	if got.code != ExitCodeUsage {
		t.Fatalf("exit code = %d", got.code)
	}
	if got.stdout != "" {
		t.Fatalf("stdout = %q", got.stdout)
	}
	for _, want := range []string{"Eats lollipops.", "USAGE", "first", "second", "COMMANDS", "eat", "examples", "Run 'test-app --help'"} {
		if !strings.Contains(got.stderr, want) {
			t.Errorf("concise help missing %q:\n%s", want, got.stderr)
		}
	}
	if strings.Contains(got.stderr, "third") || strings.Contains(got.stderr, "OPTIONS") {
		t.Errorf("concise help should be short:\n%s", got.stderr)
	}

	got = runTestMain(t, []string{"--quiet"}, commands)
	if got.code != ExitCodeUsage || !strings.Contains(got.stderr, "USAGE") {
		t.Errorf("only global flags should behave like no args: %d %q", got.code, got.stderr)
	}
}

func TestMainTopLevelHelp(t *testing.T) {
	commands := map[string]*testCommand{"eat": {name: "eat"}, "drink": {name: "drink"}}

	got := runTestMain(t, []string{"--help"}, commands,
		WithDescription("Eats lollipops."),
		WithDocsURL("https://example.com/docs"),
		WithSupportURL("https://example.com/support"),
	)
	if got.code != 0 {
		t.Fatalf("exit code = %d", got.code)
	}
	for _, want := range []string{"Eats lollipops.", "USAGE", "COMMANDS", "the drink command", "the eat command", "OPTIONS", "--version", "Read the docs at https://example.com/docs\n", "https://example.com/support"} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("help missing %q:\n%s", want, got.stdout)
		}
	}
	if strings.Index(got.stdout, "drink") > strings.Index(got.stdout, "eat ") {
		t.Error("commands should be sorted")
	}
}

func TestMainExamplesCommand(t *testing.T) {
	commands := map[string]*testCommand{
		"eat":   {name: "eat", examples: []Example{{Description: "eat one", Command: "test-app eat"}}},
		"drink": {name: "drink", examples: []Example{{Description: "drink one", Command: "test-app drink"}}},
	}

	got := runTestMain(t, []string{"examples"}, commands)
	if got.code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", got.code, got.stderr)
	}
	for _, want := range []string{"DRINK", "drink one", "EAT", "eat one"} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("examples missing %q:\n%s", want, got.stdout)
		}
	}
	if strings.Index(got.stdout, "DRINK") > strings.Index(got.stdout, "EAT") {
		t.Error("examples should be sorted by command")
	}
}

func TestMainNestedUnknownCommand(t *testing.T) {
	commands := map[string]*testCommand{
		"lollipop eat":  {name: "lollipop eat"},
		"lollipop list": {name: "lollipop list"},
	}

	got := runTestMain(t, []string{"lollipop", "eatt"}, commands)
	if got.code != ExitCodeUnknownCommand || !strings.Contains(got.stderr, `Did you mean "eat"?`) {
		t.Fatalf("code = %d, stderr = %q", got.code, got.stderr)
	}

	got = runTestMain(t, []string{"lollipop", "eat"}, commands)
	if got.code != 0 {
		t.Fatalf("nested command failed: %d %q", got.code, got.stderr)
	}

	got = runTestMain(t, []string{"lollipop", "--help"}, commands)
	if got.code != 0 || !strings.Contains(got.stdout, "SUBCOMMANDS") || !strings.Contains(got.stdout, "the lollipop list command") {
		t.Fatalf("parent help should list subcommands: %d %q %q", got.code, got.stdout, got.stderr)
	}

	got = runTestMain(t, []string{"lollipop"}, commands)
	if got.code != ExitCodeUsage || got.stdout != "" || !strings.Contains(got.stderr, "SUBCOMMANDS") {
		t.Fatalf("a parent without a subcommand should show its help on stderr: %d %q %q", got.code, got.stdout, got.stderr)
	}

	got = runTestMain(t, []string{"--help"}, commands)
	if !strings.Contains(got.stdout, "lollipop") {
		t.Fatalf("top-level help should list parents: %q", got.stdout)
	}
}

func TestRewriteHelpArgs(t *testing.T) {
	commands := map[string]cli.CommandFactory{"eat": nil}

	tests := []struct {
		args []string
		want []string
	}{
		{args: []string{"eat"}, want: []string{"eat"}},
		{args: []string{"help"}, want: []string{"--help"}},
		{args: []string{"help", "eat"}, want: []string{"eat", "--help"}},
		{args: []string{"help", "--quiet", "lollipop", "eat"}, want: []string{"lollipop", "eat", "--quiet", "--help"}},
	}
	for _, tt := range tests {
		if got := rewriteHelpArgs(tt.args, commands); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("rewriteHelpArgs(%v) = %v, want %v", tt.args, got, tt.want)
		}
	}

	commands["help"] = nil
	if got := rewriteHelpArgs([]string{"help", "eat"}, commands); !reflect.DeepEqual(got, []string{"help", "eat"}) {
		t.Errorf("an app help command should not be rewritten: %v", got)
	}
}

func TestSuggest(t *testing.T) {
	candidates := []string{"eat", "list", "version", "upgrade", "update"}

	tests := []struct {
		input string
		want  []string
	}{
		{input: "eatt", want: []string{"eat"}},
		{input: "ea", want: []string{"eat"}},
		{input: "lsit", want: []string{"list"}},
		{input: "verison", want: []string{"version"}},
		{input: "updte", want: []string{"update"}},
		{input: "upgrad", want: []string{"upgrade"}},
		{input: "zzzzzz", want: []string{}},
	}
	for _, tt := range tests {
		if got := Suggest(tt.input, candidates); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Suggest(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"kitten", "sitting", 3},
		{"eat", "eat", 0},
		{"eat", "eta", 2},
	}
	for _, tt := range tests {
		if got := levenshtein(tt.a, tt.b); got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestDidYouMean(t *testing.T) {
	quote := func(s string) string { return `"` + s + `"` }
	if got := didYouMean(nil, quote); got != "" {
		t.Errorf("no suggestions: %q", got)
	}
	if got := didYouMean([]string{"a"}, quote); got != `Did you mean "a"?` {
		t.Errorf("one: %q", got)
	}
	if got := didYouMean([]string{"a", "b"}, quote); got != `Did you mean "a" or "b"?` {
		t.Errorf("two: %q", got)
	}
	if got := didYouMean([]string{"a", "b", "c", "d"}, quote); got != `Did you mean "a", "b" or "c"?` {
		t.Errorf("many: %q", got)
	}
}

func TestWrapText(t *testing.T) {
	got := wrapText("one two three four five", 9)
	if got != "one two\nthree\nfour five" {
		t.Errorf("wrapText = %q", got)
	}
	if got := wrapText("keep\n  indented line that is long", 5); !strings.Contains(got, "  indented line that is long") {
		t.Errorf("indented lines should be preformatted: %q", got)
	}
	if got := wrapText("no wrap", 0); got != "no wrap" {
		t.Errorf("width 0 = %q", got)
	}
}
