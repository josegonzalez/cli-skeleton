package commands

import (
	"fmt"
	"os"
	"strconv"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

type GlobalCommand struct {
	command.Meta
	GlobalFlagCommand
}

func (c *GlobalCommand) Name() string {
	return "global"
}

func (c *GlobalCommand) Synopsis() string {
	return "Global command that prints the values of the global flags"
}

func (c *GlobalCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *GlobalCommand) Examples() map[string]string {
	appName := os.Getenv("CLI_APP_NAME")
	return map[string]string{
		"Prints the values of the global flags": fmt.Sprintf("%s %s", appName, c.Name()),
	}
}

func (c *GlobalCommand) Arguments() []command.Argument {
	args := []command.Argument{}
	return args
}

func (c *GlobalCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *GlobalCommand) ParsedArguments(args []string) (map[string]command.Argument, error) {
	return command.ParseArguments(args, c.Arguments())
}

func (c *GlobalCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), command.FlagSetClient)
	c.GlobalFlags(f)
	return f
}

func (c *GlobalCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(command.FlagSetClient),
		c.AutocompleteGlobalFlags(),
		complete.Flags{},
	)
}

func (c *GlobalCommand) Run(args []string) int {
	flags := c.FlagSet()
	flags.Usage = func() { c.Ui.Output(c.Help()) }
	if err := flags.Parse(args); err != nil {
		c.Ui.Error(err.Error())
		c.Ui.Error(command.CommandErrorText(c))
		return 1
	}

	_, err := c.ParsedArguments(flags.Args())
	if err != nil {
		c.Ui.Error(err.Error())
		c.Ui.Error(command.CommandErrorText(c))
		return 1
	}

	c.Ui.Output(fmt.Sprintf("Global bool value: %s", strconv.FormatBool(c.global)))
	c.Ui.Output(fmt.Sprintf("Global string value: %s", c.globalValue))

	return 0
}
