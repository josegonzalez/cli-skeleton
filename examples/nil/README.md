# nil

An example "nil" tool for the `cli-skeleton` project. It does nothing, but provides a `nil.go` command file that can be used for scaffolding commands in other projects.

## Building

```shell
# substitute the version number as desired
go build -ldflags "-X main.Version=0.1.0
```

## Usage

```
Usage: nil [--version] [--help] <command> [<args>]

Available commands are:
    nil        Nil command
    version    Return the version of the binary
```
