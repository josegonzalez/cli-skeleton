package command

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
	flag "github.com/spf13/pflag"
)

// testCommand is a minimal command used to exercise Main.
type testCommand struct {
	Meta

	name    string
	flags   FlagSetFlags
	extra   func(*flag.FlagSet)
	args    []Argument
	run      func(c *testCommand, arguments map[string]Argument) int
	examples []Example
}

func (c *testCommand) Name() string                { return c.name }
func (c *testCommand) Synopsis() string            { return "the " + c.name + " command" }
func (c *testCommand) Help() string                { return CommandHelp(c) }
func (c *testCommand) Arguments() []Argument       { return append([]Argument{}, c.args...) }
func (c *testCommand) Examples() []Example         { return c.examples }

func (c *testCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.name, c.flags)
	if c.extra != nil {
		c.extra(f)
	}
	return f
}

func (c *testCommand) Run(args []string) int {
	return Execute(c, args, func(arguments map[string]Argument) error {
		if c.run == nil {
			return nil
		}
		if code := c.run(c, arguments); code != 0 {
			return &ExitStatus{Code: code}
		}
		return nil
	})
}

type mainResult struct {
	code   int
	stdout string
	stderr string
}

func runTestMain(t *testing.T, args []string, commands map[string]*testCommand, opts ...Option) mainResult {
	t.Helper()
	return runTestMainWithInput(t, "", args, commands, opts...)
}

func runTestMainWithInput(t *testing.T, input string, args []string, commands map[string]*testCommand, opts ...Option) mainResult {
	t.Helper()
	return runTestMainFull(t, input, args, commands, nil, opts...)
}

// runTestMainWithFactories runs Main with test commands and extra commands
// built from factories.
func runTestMainWithFactories(t *testing.T, args []string, commands map[string]*testCommand, extra map[string]func(Meta) Command, opts ...Option) mainResult {
	t.Helper()
	return runTestMainFull(t, "", args, commands, extra, opts...)
}

func runTestMainFull(t *testing.T, input string, args []string, commands map[string]*testCommand, extra map[string]func(Meta) Command, opts ...Option) mainResult {
	t.Helper()
	clearColorEnv(t)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	opts = append(opts, withStreams(strings.NewReader(input), stdout, stderr))

	commandsFunc := func(ctx context.Context, meta Meta) map[string]cli.CommandFactory {
		factories := map[string]cli.CommandFactory{}
		for name, c := range commands {
			c := c
			factories[name] = func() (cli.Command, error) {
				c.Meta = meta
				return c, nil
			}
		}
		for name, build := range extra {
			build := build
			factories[name] = func() (cli.Command, error) {
				return build(meta).(cli.Command), nil
			}
		}
		return factories
	}

	code := Main(context.Background(), "test-app", "1.2.3", args, commandsFunc, opts...)
	return mainResult{code: code, stdout: stdout.String(), stderr: stderr.String()}
}

func TestMainRunsCommand(t *testing.T) {
	commands := map[string]*testCommand{
		"eat": {name: "eat", flags: FlagSetClient, run: func(c *testCommand, _ map[string]Argument) int {
			c.Ui.Output("ate " + c.AppName() + " " + c.Version())
			c.Ui.Info("info")
			return 3
		}},
	}

	got := runTestMain(t, []string{"eat"}, commands)
	if got.code != 3 {
		t.Errorf("exit code = %d", got.code)
	}
	if got.stdout != "ate test-app 1.2.3\n" {
		t.Errorf("stdout = %q", got.stdout)
	}
	if got.stderr != "info\n" {
		t.Errorf("stderr = %q", got.stderr)
	}
}

func TestMainQuietBeforeSubcommand(t *testing.T) {
	commands := map[string]*testCommand{
		"eat": {name: "eat", flags: FlagSetClient, run: func(c *testCommand, _ map[string]Argument) int {
			c.Ui.Output("output")
			c.Ui.Info("info")
			if !c.IsQuiet() {
				return 1
			}
			return 0
		}},
	}

	for _, args := range [][]string{{"--quiet", "eat"}, {"-q", "eat"}, {"eat", "-q"}} {
		got := runTestMain(t, args, commands)
		if got.code != 0 {
			t.Errorf("%v: exit code = %d, stderr = %q", args, got.code, got.stderr)
		}
		if got.stdout != "output\n" || got.stderr != "" {
			t.Errorf("%v: stdout = %q, stderr = %q", args, got.stdout, got.stderr)
		}
	}
}

func TestMainGlobalFlagsBeforeSubcommand(t *testing.T) {
	var global bool
	var globalString string
	commands := map[string]*testCommand{
		"eat": {
			name:  "eat",
			flags: FlagSetClient,
			extra: func(f *flag.FlagSet) {
				f.BoolVar(&global, "global", false, "")
				f.StringVarP(&globalString, "global-string", "g", "", "")
			},
		},
	}

	got := runTestMain(t, []string{"--global", "--global-string", "value", "eat"}, commands, WithGlobalFlags(testGlobalFlags))
	if got.code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", got.code, got.stderr)
	}
	if !global || globalString != "value" {
		t.Fatalf("global flags not applied: %v %q", global, globalString)
	}
}

func TestMainUnknownFlagBeforeSubcommand(t *testing.T) {
	commands := map[string]*testCommand{"eat": {name: "eat", flags: FlagSetClient}}

	got := runTestMain(t, []string{"--unknown", "eat"}, commands)
	if got.code == 0 {
		t.Fatalf("expected failure, stdout = %q", got.stdout)
	}
}

func TestMainHelpGoesToStdout(t *testing.T) {
	commands := map[string]*testCommand{"eat": {name: "eat", flags: FlagSetClient}}

	got := runTestMain(t, []string{"eat", "--help"}, commands)
	if got.code != 0 {
		t.Errorf("exit code = %d", got.code)
	}
	if !strings.Contains(got.stdout, "test-app eat") || got.stderr != "" {
		t.Errorf("stdout = %q, stderr = %q", got.stdout, got.stderr)
	}

	got = runTestMain(t, []string{"--help"}, commands)
	if got.code != 0 || !strings.Contains(got.stdout, "eat") {
		t.Errorf("top-level help: code = %d, stdout = %q", got.code, got.stdout)
	}

	got = runTestMain(t, []string{"help"}, commands)
	if got.code != 0 || !strings.Contains(got.stdout, "COMMANDS") {
		t.Errorf("help: code = %d, stdout = %q", got.code, got.stdout)
	}

	got = runTestMain(t, []string{"help", "eat"}, commands)
	if got.code != 0 || !strings.Contains(got.stdout, "test-app eat") || strings.Contains(got.stdout, "COMMANDS") {
		t.Errorf("help eat: code = %d, stdout = %q", got.code, got.stdout)
	}
}

func TestMainVersion(t *testing.T) {
	commands := map[string]*testCommand{}

	got := runTestMain(t, []string{"--version"}, commands)
	if got.code != 0 || got.stdout != "1.2.3\n" {
		t.Errorf("code = %d, stdout = %q", got.code, got.stdout)
	}
}

func TestMainUnknownCommand(t *testing.T) {
	commands := map[string]*testCommand{"eat": {name: "eat"}}

	got := runTestMain(t, []string{"drink"}, commands)
	if got.code != ExitCodeUnknownCommand {
		t.Errorf("exit code = %d", got.code)
	}
	if got.stdout != "" || !strings.Contains(got.stderr, `Unknown command "drink"`) {
		t.Errorf("stdout = %q, stderr = %q", got.stdout, got.stderr)
	}
	if strings.Contains(got.stderr, "Did you mean") {
		t.Errorf("no suggestion expected: %q", got.stderr)
	}

	got = runTestMain(t, []string{"eatt"}, commands)
	if got.code != ExitCodeUnknownCommand || !strings.Contains(got.stderr, `Did you mean "eat"?`) {
		t.Errorf("code = %d, stderr = %q", got.code, got.stderr)
	}
}

func TestMainWithUiAppliesSettings(t *testing.T) {
	commands := map[string]*testCommand{
		"eat": {name: "eat", flags: FlagSetClient, run: func(c *testCommand, _ map[string]Argument) int {
			c.Ui.Info("info")
			return 0
		}},
	}

	custom := func(ui cli.Ui) cli.Ui {
		return &cli.PrefixedUi{InfoPrefix: "> ", Ui: ui}
	}

	got := runTestMain(t, []string{"eat"}, commands, WithUi(custom))
	if got.stderr != "> info\n" {
		t.Errorf("stderr = %q", got.stderr)
	}

	got = runTestMain(t, []string{"eat", "--quiet"}, commands, WithUi(custom))
	if got.stderr != "" {
		t.Errorf("quiet should apply to custom UIs: %q", got.stderr)
	}
}

func TestMainZerologUiKeepsType(t *testing.T) {
	var isZerolog bool
	commands := map[string]*testCommand{
		"eat": {name: "eat", flags: FlagSetClient, run: func(c *testCommand, _ map[string]Argument) int {
			_, isZerolog = c.Ui.(*ZerologUi)
			return 0
		}},
	}

	runTestMain(t, []string{"eat"}, commands, WithUi(func(ui cli.Ui) cli.Ui {
		return HumanZerologUiWithFields(ui, nil)
	}))
	if !isZerolog {
		t.Fatal("a ZerologUi passed to WithUi should reach commands unwrapped")
	}
}

func TestMoveGlobalFlags(t *testing.T) {
	commands := map[string]cli.CommandFactory{
		"eat":          nil,
		"lollipop":     nil,
		"lollipop eat": nil,
	}
	known := globalFlagSet(testGlobalFlags)

	tests := []struct {
		args []string
		want []string
	}{
		{args: []string{"eat"}, want: []string{"eat"}},
		{args: []string{"eat", "--quiet"}, want: []string{"eat", "--quiet"}},
		{args: []string{"--quiet", "eat", "slowly"}, want: []string{"eat", "--quiet", "slowly"}},
		{args: []string{"-qd", "eat"}, want: []string{"eat", "-qd"}},
		{args: []string{"--global-string", "x", "eat"}, want: []string{"eat", "--global-string", "x"}},
		{args: []string{"--global-string=x", "eat"}, want: []string{"eat", "--global-string=x"}},
		{args: []string{"-gx", "eat"}, want: []string{"eat", "-gx"}},
		{args: []string{"-g", "x", "eat"}, want: []string{"eat", "-g", "x"}},
		{args: []string{"--quiet", "lollipop", "eat", "fast"}, want: []string{"lollipop", "eat", "--quiet", "fast"}},
		{args: []string{"--quiet", "lollipop", "--debug"}, want: []string{"lollipop", "--quiet", "--debug"}},
		{args: []string{"--quiet", "--help"}, want: []string{"--help"}},
		{args: []string{"--quiet"}, want: []string{}},
		{args: []string{"--version", "--quiet"}, want: []string{"--version", "--quiet"}},
		{args: []string{"--unknown", "eat"}, want: []string{"--unknown", "eat"}},
		{args: []string{"--quiet", "drink"}, want: []string{"drink", "--quiet"}},
	}

	for _, tt := range tests {
		got := moveGlobalFlags(tt.args, known, commands)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("moveGlobalFlags(%v) = %v, want %v", tt.args, got, tt.want)
		}
	}
}
