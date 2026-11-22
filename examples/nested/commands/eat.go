package commands

import (
	"fmt"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

const eatFlags = command.FlagSetClient | command.FlagSetForce | command.FlagSetDryRun

type EatCommand struct {
	command.Meta

	count int
}

func (c *EatCommand) Name() string {
	return "lollipop eat"
}

func (c *EatCommand) Synopsis() string {
	return "Eat lollipops from the jar"
}

func (c *EatCommand) Description() string {
	return "Eating a lollipop removes it from the jar, so you are asked to confirm first. Pass --force to skip the question, or --dry-run to see what would be eaten."
}

func (c *EatCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *EatCommand) Examples() []command.Example {
	appName := c.AppName()
	return []command.Example{
		{Description: "Eat a cherry lollipop", Command: appName + " " + c.Name() + " cherry"},
		{Description: "Eat two grape lollipops without being asked", Command: appName + " " + c.Name() + " grape --count 2 --force"},
		{Description: "See what would be eaten", Command: appName + " " + c.Name() + " lemon --dry-run", Output: "Would eat 1 lemon lollipop(s)"},
	}
}

func (c *EatCommand) Arguments() []command.Argument {
	return []command.Argument{{
		Name:        "flavor",
		Description: "the flavor of lollipop to eat",
		Type:        command.ArgumentString,
		Prompt:      "Which flavor would you like?",
		Validate:    validateFlavor,
	}}
}

func (c *EatCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictSet(flavors()...)
}

func (c *EatCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), eatFlags)
	f.IntVar(&c.count, "count", 1, "number of lollipops to eat")
	c.EnvFlag(f, "count")
	return f
}

func (c *EatCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(eatFlags),
		complete.Flags{"--count": complete.PredictAnything},
	)
}

func (c *EatCommand) Run(args []string) int {
	return command.Execute(c, args, func(arguments map[string]command.Argument) error {
		return c.eat(arguments["flavor"].StringValue())
	})
}

func (c *EatCommand) eat(flavor string) error {
	if c.count < 1 {
		return command.NewUsageError("--count must be at least 1")
	}

	if c.IsDryRun() {
		c.Ui.Output(fmt.Sprintf("Would eat %d %s lollipop(s)", c.count, flavor))
		return nil
	}

	ok, err := c.Confirm(fmt.Sprintf("Eat %d %s lollipop(s)?", c.count, flavor))
	if err != nil {
		return err
	}
	if !ok {
		c.Ui.Info("No lollipops were eaten.")
		return nil
	}

	c.Success(fmt.Sprintf("Ate %d %s lollipop(s)", c.count, flavor))
	c.Suggest(c.AppName() + " lollipop list")
	return nil
}

// MunchCommand is the old name of EatCommand, kept so existing scripts keep
// working.
type MunchCommand struct {
	EatCommand
}

func (c *MunchCommand) Name() string {
	return "lollipop munch"
}

func (c *MunchCommand) Deprecated() string {
	return fmt.Sprintf("use %q instead; it will be removed in a future release", c.AppName()+" lollipop eat")
}

func (c *MunchCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *MunchCommand) Run(args []string) int {
	return command.Execute(c, args, func(arguments map[string]command.Argument) error {
		return c.eat(arguments["flavor"].StringValue())
	})
}
