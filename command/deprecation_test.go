package command

import (
	"strings"
	"testing"

	flag "github.com/spf13/pflag"
)

type deprecatedCommand struct {
	testCommand
}

func (c *deprecatedCommand) Deprecated() string {
	return `use "eat" instead`
}

func (c *deprecatedCommand) Run(args []string) int {
	return Execute(c, args, func(arguments map[string]Argument) error {
		if c.run != nil {
			c.run(&c.testCommand, arguments)
		}
		return nil
	})
}

func TestDeprecateFlag(t *testing.T) {
	var count int
	c := &testCommand{name: "eat", flags: FlagSetClient}
	c.extra = func(f *flag.FlagSet) {
		f.IntVar(&count, "count", 1, "number of lollipops")
		f.IntVar(&count, "number", 1, "number of lollipops")
		DeprecateFlag(f, "number", "count")
	}
	commands := map[string]*testCommand{"eat": c}

	got := runTestMain(t, []string{"eat", "--number", "3"}, commands)
	if got.code != 0 || count != 3 {
		t.Fatalf("a deprecated flag should keep working: code = %d, count = %d", got.code, count)
	}
	if !strings.Contains(got.stderr, "Flag --number has been deprecated, use --count instead; --number will be removed in a future release") {
		t.Fatalf("stderr = %q", got.stderr)
	}

	got = runTestMain(t, []string{"eat", "--count", "3"}, commands)
	if strings.Contains(got.stderr, "deprecated") {
		t.Fatalf("the warning should only appear when the flag is used: %q", got.stderr)
	}

	got = runTestMain(t, []string{"eat", "--help"}, commands)
	if strings.Contains(got.stdout, "--number") {
		t.Fatalf("deprecated flags should be hidden from help:\n%s", got.stdout)
	}

	f := flag.NewFlagSet("eat", flag.ContinueOnError)
	f.Bool("old", false, "")
	DeprecateFlag(f, "old", "")
	if got := f.Lookup("old").Deprecated; got != "it will be removed in a future release" {
		t.Fatalf("message without a replacement = %q", got)
	}
}

func TestDeprecatedCommand(t *testing.T) {
	var ran bool
	old := &testCommand{name: "munch", run: func(*testCommand, map[string]Argument) int {
		ran = true
		return 0
	}}

	commandsFunc := map[string]*testCommand{"eat": {name: "eat"}}
	got := runTestMainWithFactories(t, []string{"munch"}, commandsFunc, map[string]func(Meta) Command{
		"munch": func(meta Meta) Command {
			c := &deprecatedCommand{testCommand: *old}
			c.Meta = meta
			c.run = old.run
			return c
		},
	})
	if got.code != 0 || !ran {
		t.Fatalf("a deprecated command should keep working: %d %q", got.code, got.stderr)
	}
	if !strings.Contains(got.stderr, `Command "munch" is deprecated: use "eat" instead`) {
		t.Fatalf("stderr = %q", got.stderr)
	}

	got = runTestMainWithFactories(t, []string{"--help"}, commandsFunc, map[string]func(Meta) Command{
		"munch": func(meta Meta) Command {
			c := &deprecatedCommand{testCommand: *old}
			c.Meta = meta
			return c
		},
	})
	if strings.Contains(got.stdout, "munch") {
		t.Fatalf("deprecated commands should be hidden from the command list:\n%s", got.stdout)
	}
}

func TestAliases(t *testing.T) {
	var ran int
	commands := map[string]*testCommand{
		"remove": {name: "remove", run: func(*testCommand, map[string]Argument) int {
			ran++
			return 0
		}},
	}
	aliases := WithAliases(map[string]string{"rm": "remove", "del": "missing"})

	got := runTestMain(t, []string{"rm"}, commands, aliases)
	if got.code != 0 || ran != 1 {
		t.Fatalf("the alias should run its command: code = %d, stderr = %q", got.code, got.stderr)
	}

	got = runTestMain(t, []string{"--help"}, commands, aliases)
	if strings.Contains(got.stdout, "  rm ") {
		t.Fatalf("aliases should be hidden from the command list:\n%s", got.stdout)
	}

	got = runTestMain(t, []string{"remove", "--help"}, commands, aliases)
	if !strings.Contains(got.stdout, "ALIASES\n  rm") {
		t.Fatalf("command help should list its aliases:\n%s", got.stdout)
	}

	got = runTestMain(t, []string{"del"}, commands, aliases)
	if got.code != ExitCodeUnknownCommand {
		t.Fatalf("an alias for a missing command should be ignored: %d", got.code)
	}

	got = runTestMain(t, []string{"rem"}, commands, aliases)
	if got.code != ExitCodeUnknownCommand {
		t.Fatalf("commands should never be matched by prefix: %d", got.code)
	}

	got = runTestMain(t, []string{"rn"}, commands, aliases)
	if strings.Contains(got.stderr, `"rm"`) {
		t.Fatalf("hidden aliases should not be suggested: %q", got.stderr)
	}
}
