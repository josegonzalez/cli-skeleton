# human-readable-logging

An example cli tool for the `cli-skeleton` project. It switches the logger to use zerolog with "human-readable" for output. Other than configuring zerolog as the logger, it is equivalent to the `hello-world` example, so see the [`hello-world` tutorial](../hello-world/README.md) for how to build the cli tool and add commands.

## Building

```shell
# substitute the version number as desired
go build -ldflags "-X main.Version=0.1.0"
```

## Usage

```
USAGE
  human-readable-logging <command> [flags]

COMMANDS
  completion   Print or install the shell completion script
  eat          Eats one or more lollipops
  examples     Show examples for every command
  version      Return the version of the binary

OPTIONS
  -h, --help      show help for the app or a command
  -v, --version   show the version

Run 'human-readable-logging help <command>' or 'human-readable-logging <command> --help' for more information on a command.
```

## Using human-readable logging

The human-readable logger is configured with the `command.WithUi` option in `main.go`:

```go
func main() {
  os.Exit(command.Main(context.Background(), AppName, Version, os.Args[1:], Commands,
    command.WithUi(func(ui cli.Ui) cli.Ui {
      return command.HumanZerologUiWithFields(ui, make(map[string]interface{}, 0))
    }),
  ))
}
```

Some other logging rules:

- Warn messages are prefixed with ` ?     ` and are _yellow_.
- Trace messages are prefixed with ` ++    ` and are both _magenta_ and **bold**.
- Debug messages are prefixed with ` +     ` and are both _magenta_ and **bold**, and are only shown under `--debug`.
- Error, Fatal, and Panic messages are prefixed with ` !     ` and are both _red_ and **bold**
- The underlying logger has `LogHeader1` and `LogHeader2` functions which have their output prefixed with `=====> ` and `-----> `, respectively.
- Any messages logged via `.Info()` are prefixed with `       `. `.Output()` log messages are left as is.

The logger also follows the same rules as the default UI:

- `Output()` messages go to stdout, while info, header, warning and error messages go to stderr.
- Color is only used when the stream is a terminal, and `NO_COLOR` and `--no-color` are respected.
- `--quiet` drops `Info()` messages and headers.
- Timestamps are only shown under `--debug`.

The underlying logger can be retrieved in a command like so:

```go
logger, ok := c.Ui.(*command.ZerologUi)
if !ok {
  return errors.New("unable to fetch logger from cli")
}

logger.LogHeader1("Eating lollipops")
```
