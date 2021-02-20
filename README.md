# cli-skeleton

An opinionated framework for building golang cli tools on top of [mitchellh/cli](https://github.com/mitchellh/cli).

## Why

While `mitchellh/cli` gives quite a bit of code to allow folks to build cli tools on top of it, it does not provide enough structure to allow folks to get started quickly. This project aims to fill that void by implementing a skeleton based upon those provided by the hashicorp suite of tools.

## Usage

Create a `main.go` with the following contents:

```go
package main

import (
  "fmt"
  "os"

  "github.com/josegonzalez/cli-skeleton/command"
  "github.com/mitchellh/cli"
)

// The name of the cli tool
var AppName = "cli-tool"

// Holds the version
var Version string

func main() {
  os.Exit(Run(os.Args[1:]))
}

// Executes the specified subcommand
func Run(args []string) int {
  commandMeta, ui := command.SetupRun(AppName, Version, args)
  c := cli.NewCLI(AppName, Version)
  c.Args = os.Args[1:]
  c.Commands = command.Commands(commandMeta, ui, Subcommands)
  exitCode, err := c.Run()
  if err != nil {
    fmt.Fprintf(os.Stderr, "Error executing CLI: %s\n", err.Error())
    return 1
  }

  return exitCode
}

// Returns a list of implemented subcommands
func Subcommands(meta command.Meta) map[string]cli.CommandFactory {
  return map[string]cli.CommandFactory{
    "version": func() (cli.Command, error) {
      return &command.VersionCommand{Meta: meta}, nil
    },
  }
}
```
