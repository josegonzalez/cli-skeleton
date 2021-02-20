package command

import (
	"fmt"
	"os"

	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

type VersionCommand struct {
	Meta
}

func (c *VersionCommand) Help() string {
	return CommandHelp(c)
}

func (c *VersionCommand) Arguments() []Argument {
	args := []Argument{}
	return args
}

func (c *VersionCommand) AutocompleteFlags() complete.Flags {
	return MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(FlagSetClient),
		complete.Flags{},
	)
}

func (c *VersionCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *VersionCommand) Examples() map[string]string {
	appName := os.Getenv("CLI_APP_NAME")
	return map[string]string{
		"Return the version of the binary": fmt.Sprintf("%s %s", appName, c.Name()),
	}
}

func (c *VersionCommand) FlagSet() *flag.FlagSet {
	return c.Meta.FlagSet(c.Name(), FlagSetClient)
}

func (c *VersionCommand) Name() string {
	return "version"
}

func (c *VersionCommand) ParsedArguments(args []string) (map[string]Argument, error) {
	return ParseArguments(args, c.Arguments())
}

func (c *VersionCommand) Synopsis() string {
	return "Return the version of the binary"
}

func (c *VersionCommand) Run(args []string) int {
	flags := c.FlagSet()
	flags.Usage = func() { c.Ui.Output(c.Help()) }
	if err := flags.Parse(args); err != nil {
		c.Ui.Error(err.Error())
		c.Ui.Error(CommandErrorText(c))
		return 1
	}

	_, err := c.ParsedArguments(flags.Args())
	if err != nil {
		c.Ui.Error(err.Error())
		c.Ui.Error(CommandErrorText(c))
		return 1
	}

	c.Ui.Output(os.Getenv("CLI_VERSION"))

	return 0
}
