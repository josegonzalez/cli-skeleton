# hello-world

An example cli tool for the `cli-skeleton` project.

## Building

```shell
# substitute the version number as desired
go build -ldflags "-X main.Version=0.1.0"
```

## Usage

```
An example cli tool that eats lollipops.

USAGE
  hello-world <command> [flags]

EXAMPLES
  Eat a lollipop
    $ hello-world eat

  Eat two red lollipops quickly
    $ hello-world eat --count 2 --color red quickly

COMMANDS
  completion   Print or install the shell completion script
  eat          Eats one or more lollipops
  examples     Show examples for every command
  version      Return the version of the binary

OPTIONS
  -h, --help      show help for the app or a command
  -v, --version   show the version

Run 'hello-world help <command>' or 'hello-world <command> --help' for more information on a command.
```

## Implementing this tutorial

To create a new cli tool, the cli-tool will need to be initialized. This tool will be called `hello-world`:

```shell
mkdir hello-world
go mod init hello-world
```

Next, create a `main.go` with the following contents:

```go
package main

import (
  "context"
  "os"

  "github.com/josegonzalez/cli-skeleton/command"
  "github.com/mitchellh/cli"
)

// The name of the cli tool
var AppName = "hello-world"

// Holds the version
var Version string

func main() {
  os.Exit(command.Main(context.Background(), AppName, Version, os.Args[1:], Commands,
    command.WithDescription("An example cli tool that eats lollipops."),
  ))
}

// Returns a list of implemented commands
func Commands(ctx context.Context, meta command.Meta) map[string]cli.CommandFactory {
  return map[string]cli.CommandFactory{
    "version": func() (cli.Command, error) {
      return &command.VersionCommand{Meta: meta}, nil
    },
  }
}
```

`command.Main` resolves the framework flags and environment variables, builds the UI, runs the requested command and returns its exit code. It also adds the `help`, `examples` and `completion` commands, and the hidden `docs` command.

Run `go build -ldflags "-X main.Version=0.1.0"` to build the `0.1.0` version of the tool in the current directory. Running `./hello-world` without a command will now show concise help on stderr, and exit with code `2`:

```
An example cli tool that eats lollipops.

USAGE
  hello-world <command> [flags]

COMMANDS
  completion   Print or install the shell completion script
  examples     Show examples for every command
  version      Return the version of the binary

Run 'hello-world --help' for more information.
```

The `cli-skeleton` project includes a helpful `version` command that can be executed via `./hello-world version` with the following output:

```
0.1.0
```

`./hello-world version --json` also prints the Go version, platform and the commit the binary was built from.

### Adding additional commands

Adding a new subcommand is straightforward. For the example `hello-world` app, an `eat` command will be created. To start, create a `commands` directory that contains an `eat.go` file. This file should contain an `EatCommand` struct as follows:

```go
import "github.com/josegonzalez/cli-skeleton/command"

type EatCommand struct {
  command.Meta
}
```

Embedding `command.Meta` provides the UI, the context, the app name and version, and the helpers used below. `EatCommand` should implement the following interface:

```go
type Command interface {
  Arguments()         []command.Argument
  AutocompleteArgs()  complete.Predictor
  AutocompleteFlags() complete.Flags
  Examples()          []command.Example
  FlagSet()           *flag.FlagSet
  Help()              string
  Name()              string
  Run(args []string)  int
  Synopsis()          string
}
```

The following section will describe each interface function and how to implement them for the example `eat` command. Each section will include all required `import` statements. Please be sure to de-duplicate them when creating the `eat.go` file.

#### Naming the command

The `Name()` function must return the name of the command. This is used in parsing, help output, and other examples. Nested commands use their full name, such as `lollipop eat`.

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

A longer description can be added with an optional `Description()` function. It is shown in the full help output and in generated docs.

#### Help output

To start, the following boilerplate help command can be quickly added (note the `import` statement, which only needs to be included once per command file):

```go
import "github.com/josegonzalez/cli-skeleton/command"

func (c *EatCommand) Help() string {
  return command.CommandHelp(c)
}
```

As long as all the other interface functions are implemented, the `eat` command will automatically support the `--help` and `-h` flags, as well as `hello-world help eat`.

#### Help examples

> While examples are excellent, it is recommended to have 5 or fewer examples in the help output. Further examples should be sent to documentation or potentially result in splitting the command into multiple commands.

Users wishing to understand _how_ to use cli tool will want a few examples. They are shown in the order they are declared, so lead with the most common uses:

```go
import (
  "fmt"

  "github.com/josegonzalez/cli-skeleton/command"
)

func (c *EatCommand) Examples() []command.Example {
  appName := c.AppName()
  return []command.Example{
    {Description: "Eats one lollipop quickly", Command: fmt.Sprintf("%s %s quickly", appName, c.Name())},
    {Description: "Eats one lollipop slowly", Command: fmt.Sprintf("%s %s slowly", appName, c.Name())},
    {Description: "Eats two lollipops quickly", Command: fmt.Sprintf("%s %s --count 2 quickly", appName, c.Name())},
    {Description: "Eats three red lollipops", Command: fmt.Sprintf("%s %s --count 3 --color red", appName, c.Name())},
  }
}
```

An `Example` can also set `Output` to show what the command prints. Examples are a great way to help users get started with the cli tool, allowing contributors to embed further examples for common tasks without having them rot in a place far away from the actual code. `hello-world examples` prints the examples of every command.

#### Arguments

Arguments can be added by specifying an `Arguments()` function like so:

```go
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
  Name        string                   // The name of the argument
  Description string                   // An optional description of the argument
  Optional    bool                     // Whether the argument is optional or not
  Type        ArgumentType             // The type of the Argument
  Value       interface{}              // The value of the interface
  HasValue    bool                     // Whether the Argument has a value. Populated during argument parsing
  Prompt      string                   // An optional question asked when a required argument is missing
  Validate    func(value string) error // An optional check run on each value before the command runs
}
```

Valid types are `ArgumentString`, `ArgumentInt`, `ArgumentBool`, `ArgumentFloat`, `ArgumentDuration`, `ArgumentFile` (an existing file, or `-` for stdin) and `ArgumentList`, which collects the remaining arguments and must come last.

When specifying an argument in the `Arguments()` function, only the following attributes should be specified:

- Name
- Description
- Optional
- Type
- Prompt
- Validate

When a required argument with a `Prompt` is missing and stdin is a terminal, the user is asked for it. Otherwise, the command shows its concise help and exits with code `2`.

#### Argument autocompletion

By default, argument autocompletion isn't necessary, so the following function is more than enough:

```go
import "github.com/posener/complete"

func (c *EatCommand) AutocompleteArgs() complete.Predictor {
  return complete.PredictNothing
}
```

Argument autocompletion is usually not useful except for when the argument is predictable from a list, which is usually only the case when only a single argument is specified or the arguments are a list. Users enable completion with `hello-world completion --install`.

#### Flags

Flag specification is fairly straightforward. Values should be stored on the `Command` struct, and in this case would be denoted in the `EatCommand` struct specified at the top of the file. While the `flag` module is supported by `mitchellh/cli`, `cli-skeleton` uses `github.com/spf13/pflag` for a richer flag parsing experience.

```go
import (
  "github.com/josegonzalez/cli-skeleton/command"
  flag "github.com/spf13/pflag"
)

// EatCommand struct respecified for completeness of example
type EatCommand struct {
  command.Meta

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

`command.FlagSetClient` adds the framework flags every command should accept: `--no-color`, `--no-input`, `-q`/`--quiet`, `-d`/`--debug` and `--no-pager`. Other flag set bits add standard flags such as `--json`, `--plain`, `-f`/`--force` and `-n`/`--dry-run`; see the [project README](../../README.md#arguments-and-flags) for the full list.

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
      "--count": complete.PredictAnything,
      "--color": complete.PredictSet("red", "orange", "yellow", "green", "blue", "purple"),
    },
  )
}
```

#### Defining the main `Run()` codeblock

Once a command has been filled out, the only thing left is defining the `Run()` command. `command.Execute` parses the flags and arguments, then runs the given function with the parsed arguments.

The following is the `Run()` command for our example `EatCommand`.

```go
import (
  "fmt"

  "github.com/josegonzalez/cli-skeleton/command"
)

func (c *EatCommand) Run(args []string) int {
  return command.Execute(c, args, func(arguments map[string]command.Argument) error {
    name := arguments["speed"].StringValue()
    if name == "" {
      name = "normally"
    }

    c.Ui.Output(fmt.Sprintf("Eating %d %v lollipop(s) %v", c.count, c.color, name))

    return nil
  })
}
```

> Note that arguments and flags are only validated as far as their types and `Validate` functions go - any further validation is an exercise left to the developer.

`command.Execute` turns the returned error into output and an exit code:

- Invalid flags or arguments print a short error and a pointer to the help, and exit with code `2`.
- `command.NewUserError(message, hint)` prints the message, then a hint on how to fix it.
- Common errors, such as missing files or refused connections, are rewritten with a suggested fix.
- Any other error prints its message and exits with code `1`.

Primary output should be written with `c.Ui.Output()`, which goes to stdout. Messages for the user, such as progress or warnings, should be written with `c.Ui.Info()` and `c.Ui.Warn()`, which go to stderr so they never end up in a pipe.

#### Adding the command to the cli

To add the new command, modify the `Commands()` function in the `main.go` to specify the new `eat` subcommand. The following is the full content of that function, including the necessary import statements:

```go
import (
  "context"

  "hello-world/commands"

  "github.com/josegonzalez/cli-skeleton/command"
  "github.com/mitchellh/cli"
)

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
```

#### Testing

`commandtest.ValidateCommands` checks the commands against the Command Line Interface Guidelines. Add a `main_test.go` next to `main.go`:

```go
package main

import (
  "testing"

  "github.com/josegonzalez/cli-skeleton/command/commandtest"
)

func TestCommands(t *testing.T) {
  commandtest.ValidateCommands(t, AppName, Commands)
}
```

#### Building

Once everything is put together, the `go build -ldflags "-X main.Version=0.1.0"` command - with the version modified as desired - can be executed to rebuild the binary. The output of `./hello-world eat --help` is now:

```
USAGE
  hello-world eat [flags] [speed]

Eats one or more lollipops

EXAMPLES
  Eats one lollipop quickly
    $ hello-world eat quickly

  Eats one lollipop slowly
    $ hello-world eat slowly

  Eats two lollipops quickly
    $ hello-world eat --count 2 quickly

  Eats three red lollipops
    $ hello-world eat --count 3 --color red

ARGUMENTS
  [speed] string   how quickly to eat the lollipop

OPTIONS
      --color string   the color of the lollipops being eaten (default "normal")
      --count int      number of lollipops to eat (default 1)
  -d, --debug          print debugging output; DEBUG or HELLO_WORLD_DEBUG may also be set
      --no-color       disable colored output; NO_COLOR or HELLO_WORLD_NO_COLOR may also be set
      --no-input       never prompt for input; fail instead when input is required
      --no-pager       do not page long output
  -q, --quiet          only print essential output
```

If there are any errors in compilation or output, please compare with the code in this directory.

## Distributing

clig.dev recommends distributing a cli tool as a single binary, and making it easy to uninstall. Go builds a single binary for each platform, and the included [`.goreleaser.yaml`](.goreleaser.yaml) packages one per operating system and architecture with [GoReleaser](https://goreleaser.com):

```shell
goreleaser release --snapshot --clean
```

### Installing

Download the archive for your platform, extract it, and move the binary somewhere on your `PATH`:

```shell
tar -xzf hello-world_0.1.0_linux_amd64.tar.gz
sudo mv hello-world /usr/local/bin/
```

Optionally, install shell completion and man pages for the current user:

```shell
hello-world completion --install
hello-world docs --format man --install
```

### Uninstalling

Remove the binary, along with the completion script and man pages if they were installed:

```shell
hello-world completion --uninstall
rm -f ~/.local/share/man/man1/hello-world.1 ~/.local/share/man/man1/hello-world-*.1
sudo rm /usr/local/bin/hello-world
```

Logs and other state are kept in `~/.local/state/hello-world`, and can be removed as well.
