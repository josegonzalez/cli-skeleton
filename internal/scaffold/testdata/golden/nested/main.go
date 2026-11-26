package main

import (
	"context"
	"os"

	"nested/commands"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/mitchellh/cli"
)

// The name of the cli tool
var AppName = "nested"

// Holds the version
var Version string

func main() {
	os.Exit(command.Main(context.Background(), AppName, Version, os.Args[1:], Commands,
		command.WithDescription("An example cli tool with nested commands for managing lollipops."),
		command.WithExamples(
			command.Example{Description: "List the lollipops", Command: AppName + " lollipop list"},
			command.Example{Description: "Eat a cherry lollipop", Command: AppName + " lollipop eat cherry"},
		),
		command.WithAliases(map[string]string{
			"lollipop ls": "lollipop list",
		}),
		command.WithConfig(),
		command.WithSupportURL("https://github.com/josegonzalez/cli-skeleton/issues"),
		command.WithIssueURL("https://github.com/josegonzalez/cli-skeleton/issues/new"),
	))
}

// Returns a list of implemented commands
func Commands(ctx context.Context, meta command.Meta) map[string]cli.CommandFactory {
	return map[string]cli.CommandFactory{
		"lollipop eat": func() (cli.Command, error) {
			return &commands.LollipopEatCommand{Meta: meta}, nil
		},
		"lollipop list": func() (cli.Command, error) {
			return &commands.LollipopListCommand{Meta: meta}, nil
		},
		"lollipop munch": func() (cli.Command, error) {
			return &commands.LollipopMunchCommand{Meta: meta}, nil
		},
		"lollipop savor": func() (cli.Command, error) {
			return &commands.LollipopSavorCommand{Meta: meta}, nil
		},
		"version": func() (cli.Command, error) {
			return &command.VersionCommand{Meta: meta}, nil
		},
	}
}
