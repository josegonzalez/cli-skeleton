package commands

import (
	"fmt"
	"os"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

type EatCommand struct {
	command.Meta

	count int
	color string
}

func (c *EatCommand) Name() string {
	return "eat"
}

func (c *EatCommand) Synopsis() string {
	return "Eats one or more lollipops"
}

func (c *EatCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *EatCommand) Examples() map[string]string {
	appName := os.Getenv("CLI_APP_NAME")
	return map[string]string{
		"Eats one lollipop quickly":  fmt.Sprintf("%s %s quickly", appName, c.Name()),
		"Eats one lollipop slowly":   fmt.Sprintf("%s %s slowly", appName, c.Name()),
		"Eats two lollipops quickly": fmt.Sprintf("%s %s --count 2 quickly", appName, c.Name()),
		"Eats three red lollipops":   fmt.Sprintf("%s %s --count 3 --color red", appName, c.Name()),
	}
}

func (c *EatCommand) Arguments() []command.Argument {
	args := []command.Argument{}
	args = append(args, command.Argument{
		Name:        "speed",
		Description: "how quickly to eat the lollipop",
		Optional:    true,
		Type:        command.ArgumentString,
	})
	return args
}

func (c *EatCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *EatCommand) ParsedArguments(args []string) (map[string]command.Argument, error) {
	return command.ParseArguments(args, c.Arguments())
}

func (c *EatCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), command.FlagSetClient)
	f.IntVar(&c.count, "count", 1, "number of lollipops to eat")
	f.StringVar(&c.color, "color", "normal", "the color of the lollipops being eaten")
	return f
}

func (c *EatCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(command.FlagSetClient),
		complete.Flags{
			"--count": complete.PredictAnything,
			"--color": complete.PredictSet("red", "orange", "yellow", "green", "blue", "purple"),
		},
	)
}

func (c *EatCommand) Run(args []string) int {
	flags := c.FlagSet()
	flags.Usage = func() { c.Ui.Output(c.Help()) }
	if err := flags.Parse(args); err != nil {
		c.Ui.Error(err.Error())
		c.Ui.Error(command.CommandErrorText(c))
		return 1
	}

	arguments, err := c.ParsedArguments(flags.Args())
	if err != nil {
		c.Ui.Error(err.Error())
		c.Ui.Error(command.CommandErrorText(c))
		return 1
	}

	name := arguments["speed"].StringValue()
	if name == "" {
		name = "normally"
	}

	c.Ui.Output(fmt.Sprintf("Eating %d %v lollipop(s) %v", c.count, c.color, name))

	return 0
}
