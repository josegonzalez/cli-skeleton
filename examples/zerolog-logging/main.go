package main

import (
	"context"
	"os"

	"zerolog-logging/commands"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/mitchellh/cli"
)

// The name of the cli tool
var AppName = "zerolog-logging"

// Holds the version
var Version string

func main() {
	os.Exit(command.Main(context.Background(), AppName, Version, os.Args[1:], Commands,
		command.WithUi(func(ui cli.Ui) cli.Ui {
			return command.ZerologUiWithFields(ui, make(map[string]interface{}, 0))
		}),
	))
}

// Returns a list of implemented commands
func Commands(ctx context.Context, meta command.Meta) map[string]cli.CommandFactory {
	return map[string]cli.CommandFactory{
		"eat": func() (cli.Command, error) {
			return &commands.EatCommand{Meta: meta}, nil
		},
		"version": func() (cli.Command, error) {
			return &command.VersionCommand{Meta: meta}, nil
		},
	}
}
