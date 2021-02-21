# global

An example "global" tool for the `cli-skeleton` project. It provides a `global.go` command file that implements global flags

## Building

```shell
go build
```

## Usage

```
Usage: nil [--version] [--help] <command> [<args>]

Available commands are:
    nil        Nil command
    version    Return the version of the binary
```

## Implementation

> All examples will include the relevant imports

To implement global flags, create a file `commands/flags.go`. This should contain a struct that contains your global flag properties:

```go
type GlobalFlagCommand struct {
  global      bool
  globalValue string
}
```

In addition to the struct, the `GeneralFlagUsage()` function must be implemented on that struct. This will output help text for the global flags. Failure to include this function will result in omitting the global flags from help output.

```go
func (c *GlobalFlagCommand) GeneralFlagUsage() string {
  return `
  --global
    A bool global flag.


  --global-value <value>
    A string global flag.
`
}
```

Next, a `GlobalFlags` function should be declared. This function assigns the values of the global flags to the aforementioned struct.

```go
import (
  flag "github.com/spf13/pflag"
)

func (c *GlobalFlagCommand) GlobalFlags(f *flag.FlagSet) {
  f.BoolVar(&c.global, "global", false, "a bool global flag")
  f.StringVar(&c.globalValue, "global-value", "", "a string global flag")
}
```

To autocomplete the global flags, implement a `AutocompleteGlobalFlags()` function. This function follows the same rules as the `AutocompleteFlags()` function normally implemented on a command

```go
import (
	"github.com/posener/complete"
)

func (c *GlobalFlagCommand) AutocompleteGlobalFlags() complete.Flags {
	return complete.Flags{
		"--global":        complete.PredictNothing,
		"--global-string": complete.PredictAnything,
	}
}
```

Lastly, there should be three modifications to each of the cli tool's commands. The first is to include `GlobalFlagCommand` in the command struct like so:

```go
import (
  "github.com/josegonzalez/cli-skeleton/command"
)

type GlobalCommand struct {
  command.Meta
  GlobalFlagCommand
}
```

The other change is to include the flags in the command `FlagSet()` function. This can be done via the wrapper `GlobalFlags()` function implemented above.

```go
import (
  "github.com/josegonzalez/cli-skeleton/command"
  flag "github.com/spf13/pflag"
)

func (c *GlobalCommand) FlagSet() *flag.FlagSet {
  f := c.Meta.FlagSet(c.Name(), command.FlagSetClient)
  c.GlobalFlags(f)
  return f
}
```

Lastly, global flags need to be added to autocompletion. This is done by including the return of `AutocompleteGlobalFlags()` in `AutocompleteFlags()`.

```go
func (c *GlobalCommand) AutocompleteFlags() complete.Flags {
  return command.MergeAutocompleteFlags(
    c.Meta.AutocompleteFlags(command.FlagSetClient),
    c.AutocompleteGlobalFlags(),
    complete.Flags{},
  )
}
```
