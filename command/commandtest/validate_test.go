package commandtest

import (
	"context"
	"strings"
	"testing"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/mitchellh/cli"
	flag "github.com/spf13/pflag"
)

// lintCommand is a configurable command for the linter tests.
type lintCommand struct {
	command.Meta

	name     string
	args     []command.Argument
	examples []command.Example
	flags    func(*lintCommand, *flag.FlagSet)
}

func (c *lintCommand) Name() string                  { return c.name }
func (c *lintCommand) Synopsis() string              { return "a command" }
func (c *lintCommand) Help() string                  { return command.CommandHelp(c) }
func (c *lintCommand) Run([]string) int              { return 0 }
func (c *lintCommand) Arguments() []command.Argument { return c.args }
func (c *lintCommand) Examples() []command.Example   { return c.examples }

func (c *lintCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.name, command.FlagSetClient)
	if c.flags != nil {
		c.flags(c, f)
	}
	return f
}

func commandsOf(commands ...*lintCommand) command.CommandFunc {
	return func(ctx context.Context, meta command.Meta) map[string]cli.CommandFactory {
		factories := map[string]cli.CommandFactory{}
		for _, c := range commands {
			c := c
			factories[c.name] = func() (cli.Command, error) {
				c.Meta = meta
				return c, nil
			}
		}
		return factories
	}
}

func expectProblems(t *testing.T, problems []string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		found := false
		for _, problem := range problems {
			if strings.Contains(problem, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected a problem containing %q, got:\n%s", want, strings.Join(problems, "\n"))
		}
	}
}

func TestValidCommands(t *testing.T) {
	commands := commandsOf(
		&lintCommand{name: "eat", flags: func(c *lintCommand, f *flag.FlagSet) {
			f.Int("count", 1, "")
			f.String("password-file", "", "")
			c.EnvFlag(f, "count")
		}},
		&lintCommand{name: "lollipop list"},
		&lintCommand{name: "version"},
	)

	if problems := Problems("hello-world", commands); len(problems) != 0 {
		t.Fatalf("expected no problems, got:\n%s", strings.Join(problems, "\n"))
	}
	ValidateCommands(t, "hello-world", commands)
}

func TestAppName(t *testing.T) {
	commands := commandsOf(&lintCommand{name: "eat"})

	expectProblems(t, Problems("HelloWorld", commands), `app name "HelloWorld" should be lowercase`)
	expectProblems(t, Problems("hello--world", commands), "dashes only between words")
	expectProblems(t, Problems("convert", commands), "already a common command")
}

func TestCommandNames(t *testing.T) {
	expectProblems(t, Problems("app", commandsOf(&lintCommand{name: "Eat"})), `command "Eat"`)
	expectProblems(t, Problems("app", commandsOf(&lintCommand{name: "lollipop eat_now"})), `"eat_now" should be lowercase`)
}

func TestReservedShorthands(t *testing.T) {
	commands := commandsOf(&lintCommand{name: "eat", flags: func(c *lintCommand, f *flag.FlagSet) {
		f.BoolP("verbose", "v", false, "")
		f.BoolP("hungry", "h", false, "")
		f.StringP("port-number", "p", "", "")
		f.Bool("x", false, "")
	}})

	expectProblems(t, Problems("app", commands),
		"--verbose uses -v, which is reserved for --version",
		"--hungry uses -h, which is reserved for help",
		"--port-number uses -p, which is reserved for --port",
		"flag --x needs a descriptive long name",
	)
}

func TestFlagSetPanic(t *testing.T) {
	commands := commandsOf(&lintCommand{name: "eat", flags: func(c *lintCommand, f *flag.FlagSet) {
		f.BoolP("silent", "q", false, "")
	}})
	expectProblems(t, Problems("app", commands), `command "eat": defining its flags panicked`)
}

func TestStandardFlagShorthand(t *testing.T) {
	commands := commandsOf(&lintCommand{name: "eat", flags: func(c *lintCommand, f *flag.FlagSet) {
		f.StringP("color", "c", "", "")
	}}, &lintCommand{name: "drink"})

	commandsWithStandard := commandsOf(&lintCommand{name: "eat"})
	if problems := Problems("app", commandsWithStandard); len(problems) != 0 {
		t.Fatalf("the framework standard flags should pass: %v", problems)
	}
	if problems := Problems("app", commands); len(problems) != 0 {
		t.Fatalf("non-standard short flags are fine: %v", problems)
	}
}

func TestStandardNameWithWrongShorthand(t *testing.T) {
	commands := commandsOf(&lintCommand{name: "eat", flags: func(c *lintCommand, f *flag.FlagSet) {
		f.StringP("label", "l", "", "")
	}})
	if problems := Problems("app", commands); len(problems) != 0 {
		t.Fatalf("unexpected problems: %v", problems)
	}

	raw := flag.NewFlagSet("eat", flag.ContinueOnError)
	raw.BoolP("all", "x", false, "")
	v := &validator{flagTypes: map[string]flagType{}}
	raw.VisitAll(func(f *flag.Flag) { v.checkFlag("eat", f) })
	expectProblems(t, v.problems, "--all should use the standard short flag -a, not -x")
}

func TestFlagTypes(t *testing.T) {
	commands := commandsOf(
		&lintCommand{name: "eat", flags: func(c *lintCommand, f *flag.FlagSet) { f.Int("count", 1, "") }},
		&lintCommand{name: "drink", flags: func(c *lintCommand, f *flag.FlagSet) { f.String("count", "", "") }},
	)
	expectProblems(t, Problems("app", commands), "flag --count is a string in \"drink\" but a int in \"eat\"")
}

func TestSimilarNames(t *testing.T) {
	commands := commandsOf(&lintCommand{name: "update"}, &lintCommand{name: "upgrade"}, &lintCommand{name: "status"})
	expectProblems(t, Problems("app", commands), `commands "update" and "upgrade" are easy to confuse`)

	nested := commandsOf(&lintCommand{name: "lollipop eat"}, &lintCommand{name: "lollipop ear"}, &lintCommand{name: "candy eat"})
	problems := Problems("app", nested)
	expectProblems(t, problems, `commands "lollipop ear" and "lollipop eat"`)
	for _, problem := range problems {
		if strings.Contains(problem, "candy") {
			t.Errorf("commands under different parents should not be compared: %s", problem)
		}
	}

	expectProblems(t, Problems("app", commandsOf(&lintCommand{name: "example"})), `commands "example" and "examples"`)
}

func TestTooManyExamples(t *testing.T) {
	examples := make([]command.Example, MaxExamples+1)
	for i := range examples {
		examples[i] = command.Example{Command: "app eat"}
	}
	expectProblems(t, Problems("app", commandsOf(&lintCommand{name: "eat", examples: examples})), "has 6 examples")
}

func TestTooManyArguments(t *testing.T) {
	var args []command.Argument
	for _, name := range []string{"a", "b", "c", "d", "e"} {
		args = append(args, command.Argument{Name: name, Type: command.ArgumentString})
	}
	expectProblems(t, Problems("app", commandsOf(&lintCommand{name: "eat", args: args})), "takes 5 positional arguments")

	withList := append(append([]command.Argument{}, args[:4]...), command.Argument{Name: "rest", Type: command.ArgumentList})
	if problems := Problems("app", commandsOf(&lintCommand{name: "eat", args: withList})); len(problems) != 0 {
		t.Fatalf("a trailing list should not count: %v", problems)
	}
}

func TestSecrets(t *testing.T) {
	commands := commandsOf(&lintCommand{name: "login", flags: func(c *lintCommand, f *flag.FlagSet) {
		f.String("password", "", "")
		f.String("api-token", "", "")
		f.String("token-file", "", "")
		f.String("keyboard", "", "")
		f.String("secret-key", "", "")
		c.EnvFlag(f, "secret-key")
	}})

	problems := Problems("app", commands)
	expectProblems(t, problems,
		"--password looks like it takes a secret",
		"--api-token looks like it takes a secret",
		"--secret-key is bound to APP_SECRET_KEY",
	)
	for _, problem := range problems {
		if strings.Contains(problem, "--token-file looks") || strings.Contains(problem, "keyboard") {
			t.Errorf("unexpected problem: %s", problem)
		}
	}
}

func TestEnvNames(t *testing.T) {
	commands := commandsOf(&lintCommand{name: "eat", flags: func(c *lintCommand, f *flag.FlagSet) {
		f.Int("count", 1, "")
		c.EnvFlag(f, "count")
	}})
	if problems := Problems("9lives", commands); len(problems) != 1 || !strings.Contains(problems[0], "app name") {
		t.Fatalf("env names are sanitized, so only the app name should be reported: %v", problems)
	}
}

func TestNotACommand(t *testing.T) {
	commands := func(ctx context.Context, meta command.Meta) map[string]cli.CommandFactory {
		return map[string]cli.CommandFactory{
			"mock": func() (cli.Command, error) { return &cli.MockCommand{}, nil },
		}
	}
	expectProblems(t, Problems("app", commands), `command "mock" does not implement command.Command`)
}
