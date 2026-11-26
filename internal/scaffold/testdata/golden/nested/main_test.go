package main

import (
	"testing"

	"github.com/josegonzalez/cli-skeleton/command/commandtest"
)

func TestCommands(t *testing.T) {
	commandtest.ValidateCommands(t, AppName, Commands)
}
