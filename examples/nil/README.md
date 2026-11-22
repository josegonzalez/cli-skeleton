# nil

An example "nil" tool for the `cli-skeleton` project. It does nothing, but provides a `nil.go` command file that can be used for scaffolding commands in other projects.

## Building

```shell
# substitute the version number as desired
go build -ldflags "-X main.Version=0.1.0"
```

## Usage

```
USAGE
  nil <command> [flags]

COMMANDS
  completion   Print or install the shell completion script
  examples     Show examples for every command
  nil          Nil command
  version      Return the version of the binary

OPTIONS
  -h, --help      show help for the app or a command
  -v, --version   show the version

Run 'nil help <command>' or 'nil <command> --help' for more information on a command.
```
