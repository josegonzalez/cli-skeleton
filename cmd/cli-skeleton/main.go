// Command cli-skeleton generates new cli-skeleton apps.
package main

import (
	"context"
	"os"

	"github.com/josegonzalez/cli-skeleton/cmd/cli-skeleton/commands"
	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/mitchellh/cli"
)

// The name of the cli tool
var AppName = "cli-skeleton"

// Holds the version
var Version string

func main() {
	os.Exit(command.Main(context.Background(), AppName, Version, os.Args[1:], Commands,
		command.WithDescription("Generate new cli tools built on cli-skeleton."),
		command.WithExamples(
			command.Example{Description: "Answer questions to set up an app in the current directory", Command: AppName + " new"},
			command.Example{Description: "Set up an app from a spec", Command: AppName + " new my-app --spec my-app.yaml"},
		),
		command.WithSupportURL("https://github.com/josegonzalez/cli-skeleton/issues"),
		command.WithIssueURL("https://github.com/josegonzalez/cli-skeleton/issues/new"),
	))
}

// Returns a list of implemented commands
func Commands(ctx context.Context, meta command.Meta) map[string]cli.CommandFactory {
	return map[string]cli.CommandFactory{
		"new": func() (cli.Command, error) {
			return &commands.NewCommand{Meta: meta}, nil
		},
		"version": func() (cli.Command, error) {
			return &command.VersionCommand{Meta: meta}, nil
		},
	}
}
