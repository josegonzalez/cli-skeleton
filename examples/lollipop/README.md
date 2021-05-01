# lollipop

An example cli tool for the `cli-skeleton` project.

## Building

```shell
# substitute the version number as desired
go build -ldflags "-X main.Version=0.1.0
```

## Usage

```
Usage: lollipop [--version] [--help] <command> [<args>]

Available commands are:
    eat        Eats one or more lollipops
    version    Return the version of the binary
```

## Getting started

> The source for this example is stored in the `examples/lollipop` directory.

To create a new cli tool, the cli-tool will need to be initialized. This tool will be called `lollipop`:

```shell
mkdir lollipop
go mod init lollipop
```

Next, create a `main.go` with the following contents:

```go
package main

import (
  "fmt"
  "os"

  "github.com/josegonzalez/cli-skeleton/command"
  "github.com/mitchellh/cli"
)

// The name of the cli tool
var AppName = "lollipop"

// Holds the version
var Version string

func main() {
  os.Exit(Run(os.Args[1:]))
}

// Executes the specified command
func Run(args []string) int {
  commandMeta, ui := command.SetupRun(AppName, Version, args)
  c := cli.NewCLI(AppName, Version)
  c.Args = os.Args[1:]
  c.Commands = command.Commands(commandMeta, ui, Commands)
  exitCode, err := c.Run()
  if err != nil {
    fmt.Fprintf(os.Stderr, "Error executing CLI: %s\n", err.Error())
    return 1
  }

  return exitCode
}

// Returns a list of implemented commands
func Commands(meta command.Meta) map[string]cli.CommandFactory {
  return map[string]cli.CommandFactory{
    "version": func() (cli.Command, error) {
      return &command.VersionCommand{Meta: meta}, nil
    },
  }
}
```

Run `go build -ldflags "-X main.Version=0.1.0"` to build the `0.1.0` version of the tool in the current directory. Running `./lollipop` will now show the following output:

```
Usage: lollipop [--version] [--help] <command> [<args>]

Available commands are:
    version    Return the version of the binary
```

The `cli-skeleton` project includes a helpful `version` command that can be executed via `./lollipop version` with the following output:

```
0.1.0
```

### Adding additional commands

Adding a new subcommand is straightforward. For the example `lollipop` app, an `eat` command will be created. To start, create a `commands` directory that contains an `eat.go` file. This file should contain an `EatCommand` struct as follows:

```go
import "github.com/josegonzalez/cli-skeleton/command"

type EatCommand struct {
  command.Meta
}
```

`EatCommand` should implement the following interface:

```go
type Command interface {
  Arguments()                    []Argument
  AutocompleteArgs()             complete.Predictor
  AutocompleteFlags()            complete.Flags
  Examples()                     map[string]string
  FlagSet()                      *flag.FlagSet
  Help()                         string
  Name()                         string
  ParsedArguments(args []string) (map[string]Argument, error)
  Run(args []string)             int
  Synopsis()                     string
}
```

The following section will describe each interface function and how to implement them for the example `eat` command. Each section will include all required `import` statements. Please be sure to de-duplicate them when creating the `eat.go`  file.

#### Naming the command

The `Name()` function must return the name of the command. This is used in parsing, help output, and other examples.

```go
func (c *EatCommand) Name() string {
  return "eat"
}
```

#### Describing the command

A command description - or `synopsis` - is used in the help output for the function. This should ideally be 50 characters or less:

```go
func (c *EatCommand) Synopsis() string {
  return "Eats one or more lollipops"
}
```

#### Help output

To start, the following boilerplate help command can be quickly added (note the `import` statement, which only needs to be included once per command file):

```go
import "github.com/josegonzalez/cli-skeleton/command"

func (c *EatCommand) Help() string {
  return command.CommandHelp(c)
}
```

As long as all the other interface functions are implemented, the `eat` command will automatically support the `--help` and `-h` flags for help output.

#### Help examples

> While examples are excellent, it is recommended to have 5 or fewer examples in the help output. Further examples should be sent to documentation or potentially result in splitting the command into multiple commands.

Users wishing to understand _how_ to use cli tool will want a few examples. These can be easily specified like so:

```go
import (
  "fmt"
  "os"
)

func (c *EatCommand) Examples() map[string]string {
  appName := os.Getenv("CLI_APP_NAME")
  return map[string]string{
    "Eats one lollipop quickly": fmt.Sprintf("%s %s quickly", appName, c.Name()),
    "Eats one lollipop slowly": fmt.Sprintf("%s %s slowly", appName, c.Name()),
    "Eats two lollipops quickly": fmt.Sprintf("%s %s --count 2 quickly", appName, c.Name()),
    "Eats three red lollipops": fmt.Sprintf("%s %s --count 3 --color red", appName, c.Name()),
  }
}
```

Examples are a great way to help users get started with the cli tool, allowing contributors to embed further examples for common tasks without having them rot in a place far away from the actual code.

#### Arguments

Arguments can be added by specifying an `Arguments()` function like so:

```
import "github.com/josegonzalez/cli-skeleton/command"

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
```

The `Arguments()` function returns a slice of `Argument` structs. An `Argument` struct is defined as follows:

```go
type Argument struct {
  Name        string       // The name of the argument
  Description string       // An optional description of the argument
  Optional    bool         // Whether the argument is optional or not
  Type        ArgumentType // The type of the Argument. Valid types are: ArgumentString, ArgumentInt, ArgumentBool, ArgumentList
  Value       interface{}  // The value of the interface
  HasValue    bool         // A boolean that contains whether the Argument has a value. Populated during argument parsing
}
```

When specifying an argument in the `Arguments()` function, only the following attributes should be specified:

- Name
- Description
- Optional
- Type

#### Argument autocompletion

By default, argument autocompletion isn't necessary, so the following function is more than enough:

```go
import "github.com/posener/complete"

func (c *EatCommand) AutocompleteArgs() complete.Predictor {
  return complete.PredictNothing
}
```

Argument autocompletion is usually not useful except for when the argument is predictable from a list, which is usually only the case when only a single argument is specified or the arguments are a list.

#### Argument Parsing

Argument parsing involves a boilerplate function. It is not strictly necessary, but makes it easier to handle arguments within the main `Run()` function of the command.

```go
import "github.com/josegonzalez/cli-skeleton/command"

func (c *EatCommand) ParsedArguments(args []string) (map[string]command.Argument, error) {
  return command.ParseArguments(args, c.Arguments())
}
```

#### Flags

Flag specification is fairly straightforward. Values should be stored on the `Command` struct, and in this case would be denoted in the `EatCommand` struct specified at the top of the file. While the `flag` module is supported by `mitchellh/cli`, `cli-skeleton` uses `github.com/spf13/pflag` for a richer flag parsing experience.

```go
import (
  "github.com/josegonzalez/cli-skeleton/command"
  flag "github.com/spf13/pflag"
)

// EatCommand struct respecified for completeness of example
type EatCommand struct {
  Meta

  count int
  color string
}

func (c *EatCommand) FlagSet() *flag.FlagSet {
  f := c.Meta.FlagSet(c.Name(), command.FlagSetClient)
  f.IntVar(&c.count, "count", 1, "number of lollipops to eat")
  f.StringVar(&c.color, "color", "normal", "the color of the lollipops being eaten")
  return f
}
```

Flags _should_ only be used for optional arguments on the command, or when specifying an argument without a name on the command line would make it less clear as to what is being specified

#### Flag autocompletion

Flag autocompletion can help in autocompleting both the flags _and_ their potential values. While the `github.com/posener/complete` library supports a wide range of prediction capabilities, below are some simple examples.

```go
import (
  "github.com/josegonzalez/cli-skeleton/command"
  "github.com/posener/complete"
)

func (c *EatCommand) AutocompleteFlags() complete.Flags {
  return command.MergeAutocompleteFlags(
    c.Meta.AutocompleteFlags(command.FlagSetClient),
    complete.Flags{
      "--count":           complete.PredictAnything,
      "--color":           complete.PredictSet("red", "orange", "yellow", "green", "blue", "purple"),
    },
  )
}
```

#### Defining the main `Run()` codeblock

Once a command has been filled out, the only thing left is defining the `Run()` command. This is used to parse arguments and flags before actually running the command code.

The following is the `Run()` command for our example `EatCommand`.

```go
import (
  "fmt"

  "github.com/josegonzalez/cli-skeleton/command"
)

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
```

> Note that arguments and flags are not validated - this is an exercise left to the developer.

Errors are output via `c.Ui.Error()` - showing the `CommandErrorText` text as appropriate. This allows users to self-discover issues with their execution of the subcommand.

Additionally, the `Run()` command returns an integer, which represents the response code. `0` should be returned in case of success, with anything between `1` and `255` being an error state. It is recommended that users respect shell exit codes when using anything other than exit codes `0` and `1`.

#### Adding the command to the cli

To add the new command, modify the `Commands()` function in the `main.go` to specify the new `eat` subcommand. The following is the full content of that function, including the necessary import statements:

```
import (
  "lollipop/commands"

  "github.com/josegonzalez/cli-skeleton/command"
  "github.com/mitchellh/cli"
)

// Returns a list of implemented commands
func Commands(meta command.Meta) map[string]cli.CommandFactory {
  return map[string]cli.CommandFactory{
    "eat": func() (cli.Command, error) {
      return &commands.EatCommand{Meta: meta}, nil
    },
    "version": func() (cli.Command, error) {
      return &command.VersionCommand{Meta: meta}, nil
    },
  }
}
```

#### Building

Once everything is put together, the `go build -ldflags "-X main.Version=0.1.0"` command - with the version modified as desired - can be executed to rebuild the binary. The following is the new help output:

```
Usage: lollipop [--version] [--help] <command> [<args>]

Available commands are:
    eat        Eats one or more lollipops
    version    Return the version of the binary
```

If there are any errors in compilation or output, please compare with the code in `examples/lollipop`.
