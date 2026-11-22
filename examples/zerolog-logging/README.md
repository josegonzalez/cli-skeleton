# zerolog-logging

An example cli tool for the `cli-skeleton` project. It switches the logger to use [zerolog](https://github.com/rs/zerolog) for output. Other than configuring zerolog as the logger, it is equivalent to the `hello-world` example, so see the [`hello-world` tutorial](../hello-world/README.md) for how to build the cli tool and add commands.

## Building

```shell
# substitute the version number as desired
go build -ldflags "-X main.Version=0.1.0"
```

## Usage

```
USAGE
  zerolog-logging <command> [flags]

COMMANDS
  completion   Print or install the shell completion script
  eat          Eats one or more lollipops
  examples     Show examples for every command
  version      Return the version of the binary

OPTIONS
  -h, --help      show help for the app or a command
  -v, --version   show the version

Run 'zerolog-logging help <command>' or 'zerolog-logging <command> --help' for more information on a command.
```

## Using zerolog

The zerolog logger is configured with the `command.WithUi` option in `main.go`:

```go
func main() {
  os.Exit(command.Main(context.Background(), AppName, Version, os.Args[1:], Commands,
    command.WithUi(func(ui cli.Ui) cli.Ui {
      return command.ZerologUiWithFields(ui, make(map[string]interface{}, 0))
    }),
  ))
}
```

The logger follows the same rules as the default UI:

- `Output()` messages go to stdout, while `Info()`, `Warn()` and `Error()` messages go to stderr.
- Color is only used when the stream is a terminal, and `NO_COLOR` and `--no-color` are respected.
- `--quiet` drops `Info()` messages.
- Timestamps and level labels, such as `5:55PM INF`, are only shown under `--debug`, which also enables `Debug()` messages.

Prompts are passed through to the UI the logger wraps, so `--no-input` and the terminal checks still apply.

The underlying zerolog logger can be retrieved in a command like so, for example to add fields to every message:

```go
logger, ok := c.Ui.(*command.ZerologUi)
if !ok {
  return errors.New("unable to fetch logger from cli")
}

logger.Field("flavor", "cherry").Info("Eating a lollipop")
```
