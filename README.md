# cli-skeleton

An opinionated framework for building golang cli tools on top of [`mitchellh/cli`](https://github.com/mitchellh/cli).

## Why

While [`mitchellh/cli`](https://github.com/mitchellh/cli) gives quite a bit of code to allow folks to build cli tools on top of it, it does not provide enough structure to allow folks to get started quickly. This project aims to fill that void by implementing a skeleton based upon those provided by the hashicorp suite of tools.

Apps built with `cli-skeleton` follow the [Command Line Interface Guidelines](https://clig.dev/) by default. The sections below describe how each guideline is handled.

### Examples

For examples on how to perform various tasks, see the following examples:

- [`global`](examples/global): Shows how to implement "global" flags.
- [`hello-world`](examples/hello-world): The hello-world example.
- [`human-readable-logging`](examples/human-readable-logging): The hello-world example with nicer log output.
- [`nested`](examples/nested): An example cli tool with nested commands, showing tables, confirmations, prompts, progress steps, locks, configuration and more.
- [`nil`](examples/nil): An example cli tool that does nothing. Useful for copying the `nil.go` command as a command template for your own cli tools.
- [`zerolog-logging`](examples/zerolog-logging): The hello-world example with zerolog logger support.

## Getting started

### Scaffolding a new project

`cli-skeleton new` sets up a new cli tool in the current directory, or in the directory passed to it:

```shell
go install github.com/josegonzalez/cli-skeleton/cmd/cli-skeleton@latest
cli-skeleton new my-app
```

It asks for the name, Go module path, description, log output and config support of the app, and then for each command: its name, synopsis, description, standard flags, arguments, flags and examples. It then writes a `main.go`, a `main_test.go` that runs `commandtest.ValidateCommands`, one file per command under `commands/` with a `Run` method to fill in, a `go.mod` and a `.gitignore`, and runs `go get` and `go mod tidy`.

Pass `--spec` to read the same details from a YAML file, or `-` for stdin, instead of answering questions. `--save-spec` saves the answers as a spec, `--dry-run` shows what would be written, `--no-tidy` skips the go commands, and `--skeleton-path` uses a local checkout of `cli-skeleton`. Existing files are never overwritten unless `--force` is passed.

```yaml
---
name: my-app
module: github.com/me/my-app # defaults to the name
description: Manages apps.
examples:
  - description: Create an app
    command: app create my-app # what follows the app name
ui: default # default, zerolog or human-zerolog
config: true # read flag defaults with command.WithConfig
docs_url: https://example.com/docs
support_url: https://example.com/support
issue_url: https://example.com/issues/new
aliases:
  app ls: app list
global_flags: # accepted by every command, before or after its name
  - name: region
    usage: the region to use
commands:
  - name: app create
    synopsis: Create an app
    description: Creates an app and prints its URL.
    deprecated: "" # set to a message to hide the command and warn when it is used
    flag_sets: [client, force, dry-run] # defaults to client
    arguments:
      - name: name
        description: the name of the app
        type: string # string, int, bool, float, duration, file or list
        optional: false
        prompt: What should the app be called?
    argument_values: [] # offered when completing the arguments
    flags:
      - name: count
        type: int # bool, int, string, float or duration
        default: "1"
        usage: number of instances
        env: true # bind MY_APP_COUNT with Meta.EnvFlag
        values: ["1", "2", "3"] # offered when completing the value
    examples:
      - description: Create an app
        args: my-app # what follows the command name
        output: Created my-app
  - name: app list
    synopsis: List the apps
    flag_sets: [client, json, plain]
```

Aliases, URLs, global flags, deprecation, completion values, environment variables and argument prompts can only be set in a spec. Every example has a `scaffold.yaml` that generates it, apart from the bodies of the `Run` methods.

### Writing commands

A cli tool is a `main.go` that hands its commands to `command.Main`:

```go
package main

import (
  "context"
  "os"

  "github.com/josegonzalez/cli-skeleton/command"
  "github.com/mitchellh/cli"
)

var AppName = "hello-world"
var Version string

func main() {
  os.Exit(command.Main(context.Background(), AppName, Version, os.Args[1:], Commands,
    command.WithDescription("An example cli tool that eats lollipops."),
  ))
}

func Commands(ctx context.Context, meta command.Meta) map[string]cli.CommandFactory {
  return map[string]cli.CommandFactory{
    "version": func() (cli.Command, error) {
      return &command.VersionCommand{Meta: meta}, nil
    },
  }
}
```

Each command embeds `command.Meta` and runs through `command.Execute`, which parses flags and arguments, runs the command and turns its error into output and an exit code:

```go
func (c *EatCommand) Run(args []string) int {
  return command.Execute(c, args, func(arguments map[string]command.Argument) error {
    c.Ui.Output("Eating a lollipop")
    return nil
  })
}
```

See the [`hello-world`](examples/hello-world) tutorial for a complete walkthrough.

`command.Main` accepts these options:

| Option | Purpose |
| --- | --- |
| `WithDescription`, `WithExamples` | Text shown in the app help |
| `WithDocsURL`, `WithSupportURL` | Links shown at the bottom of help |
| `WithIssueURL` | Where unexpected errors link to, pre-filled with build details |
| `WithUi` | Replace the UI, such as with `command.ZerologUiWithFields` |
| `WithGlobalFlags` | App-global flags accepted before the subcommand |
| `WithConfig` | Read flag defaults from config files |
| `WithAliases` | Explicit, stable command aliases |
| `WithDefaultTimeout`, `WithDefaultPort` | Defaults of `--timeout` and `--port` |
| `WithGracePeriod` | How long a command has to stop after Ctrl+C |
| `WithTailLines` | How many output lines running steps show |

## Command line interface guidelines

### The basics

- Exit codes: `0` on success, `1` on failure, `2` for invalid flags or arguments (`command.ExitCodeUsage`), `127` for unknown commands and `130` when interrupted. Commands can return `&command.ExitStatus{Code: n}` to exit with any other code.
- Primary output goes to stdout through `Ui.Output`. Messages, warnings, errors and prompts go to stderr, so piping a command only passes on its output.

### Help

- `-h`, `--help`, `app help` and `app help <command>` show full help on stdout and exit `0`.
- Running the app without arguments, a command without its required arguments, or a parent command without a subcommand shows concise help on stderr and exits `2`.
- Help leads with usage and examples, shown in the order they are declared, followed by arguments, options, subcommands, aliases and links to the docs and support.
- Headings are bold only on a color-capable terminal, help wraps to `COLUMNS` or the terminal width, and help taller than the terminal is paged.
- Unknown commands and flags suggest the closest match. On a terminal, a single close match is offered as a question, and never run without a yes.
- The built-in `examples` command prints every example as a cheat sheet.

### Documentation

- The hidden `docs` command writes markdown or man page reference docs, built from the same text as the help output: `app docs --format markdown --output docs`.
- `app docs --format man --install` writes man pages to `~/.local/share/man/man1` after asking for confirmation, so `man app-<command>` works.
- Command help links to `<docs url>/<command-path>`, such as `/lollipop-eat`, matching the generated file names.

### Output

- Color is decided separately for stdout and stderr. It is disabled when the stream is not a terminal, when `NO_COLOR` or `<APP>_NO_COLOR` is set, when `TERM=dumb`, or with `--no-color`. `FORCE_COLOR` enables it without checking for a terminal, though `NO_COLOR` still wins.
- Only errors (red), warnings (yellow), headings and success marks use color.
- Commands can opt into `--json` and `--plain`. `OutputTable` prints aligned columns, one tab-separated record per line under `--plain`, or JSON under `--json`, and `OutputJSON` writes indented JSON.
- `-q`/`--quiet` hides messages but keeps primary output.
- `Success` reports what changed, and `Suggest` tells the user what to run next.
- `OutputPaged` pages long output through `<APP>_PAGER`, `PAGER` or `less` (with `LESS=FIRX` unless `LESS` is set), only when stdin and stdout are terminals and `--no-pager` is not passed.
- `IsInteractive`, `IsStdoutTerminal` and `IsStderrTerminal` let commands skip animations outside a terminal.
- The zerolog UIs only show timestamps and level labels under `--debug`.

### Errors

- `command.NewUserError(message, hint)` prints the message and then the hint, so the suggested fix is the last thing on screen.
- Common low-level errors, such as permission denied, missing files, refused connections, timeouts and DNS failures, are rewritten with a suggested fix by `HumanizeError`.
- `command.NewMultiError` groups errors of the same kind under one header.
- Panics and `command.Internal(err)` errors print a short message and a link to report the problem, pre-filled with the version, platform and error. Details are only shown under `-d`/`--debug`, or written to the file named by `<APP>_DEBUG_LOG`.

### Arguments and flags

`FlagSetClient` adds the framework flags every command should accept: `--no-color`, `--no-input`, `-q`/`--quiet`, `-d`/`--debug` and `--no-pager`. They can also be passed before the subcommand. Commands can opt into the standard flags from clig.dev:

| Bit | Flag | Read with |
| --- | --- | --- |
| `FlagSetJSON` | `--json` | `IsJSON` |
| `FlagSetPlain` | `--plain` | `IsPlain` |
| `FlagSetForce` | `-f`, `--force` | `IsForce`, `Confirm` |
| `FlagSetDryRun` | `-n`, `--dry-run` | `IsDryRun` |
| `FlagSetConfirm` | `--confirm` | `ConfirmName` |
| `FlagSetTimeout` | `--timeout` | `Timeout`, `TimeoutContext` |
| `FlagSetOutput` | `-o`, `--output` | `OutputWriter` |
| `FlagSetAll` | `-a`, `--all` | `IsAll` |
| `FlagSetPort` | `-p`, `--port` | `Port` |
| `FlagSetUser` | `-u`, `--user` | `User` |

`FlagSetDefault` adds all of them.

- `OpenInput` and `OpenOutput` treat `-` as stdin and stdout, and `ArgumentFile` arguments accept `-` too.
- `ReadSecret` reads secrets from a file or stdin. Never accept secrets as flag values or environment variables, since they leak into shell history, process lists and child processes; accept a flag such as `--password-file` instead.
- `command.OptionalString` defines a flag that can be turned off with `none`, and rejects empty values.
- Arguments can be strings, ints, bools, floats, durations, files or a trailing list, and can define `Validate` to check values before the command runs.

### Interactivity

- Prompts only run when stdin is a terminal and `--no-input` was not passed. Otherwise they fail with an error that says how to pass the value instead.
- `Argument.Prompt` asks for a missing required argument when possible.
- `Confirm` asks a yes or no question, defaulting to no, and is skipped with `--force`. `ConfirmName` asks the user to type the name of the resource for severe actions, and accepts `--confirm=<name>` in scripts.
- `AskSecret` does not echo what is typed.
- Ctrl+C during a prompt exits right away, restoring the terminal.
- `Editor` opens text in `VISUAL` or `EDITOR`, and `RunShell` starts the user's `SHELL`.
- `RequireStdin` shows concise help instead of waiting for input that will never be piped in.

### Subcommands

- Commands can be nested, such as `lollipop eat`. Parent commands are added automatically and list their subcommands.
- `WithAliases` registers explicit, stable aliases. Commands are never matched by prefix, and there is no catch-all default command, so new commands can be added without breaking scripts.

### Robustness

- Arguments are validated before a command runs.
- `Steps`, `Spinner` and `ProgressBar` show progress. On a terminal, a running step shows a spinner and a live tail of its recent output, and collapses to one line with its duration when it succeeds. Outside a terminal a line is printed as each step starts and ends, `<APP>_VERBOSE` and `--debug` stream every line, and `--quiet` stays silent unless a step fails. A failed step shows its output, the path of the log, which always holds the full output, and where to get help.
- `Step.Exec` runs external commands inside a step, with stdin set to the null device, and stops them along with their children when interrupted.
- `HTTPClient` honors `HTTP_PROXY`, `HTTPS_PROXY` and `NO_PROXY`, times out with `--timeout`, and identifies itself as `<app>/<version>`. `NewRequest` builds requests that are cancelled on interrupt.
- `Lock` and `LockWait` keep concurrent runs from doing the same work.
- `Countdown` gives users time to cancel before a dangerous action.

### Future-proofing

- `DeprecateFlag` keeps a flag working, hides it from help and warns only when it is used.
- Commands implementing `Deprecated() string` warn before running and are hidden from the command list.
- Human-readable output may change between releases, so scripts should use `--plain` or `--json`.

### Signals and control characters

- The first Ctrl+C or `SIGTERM` prints `Gracefully stopping... (press Ctrl+C again to force)` and cancels `Meta.Context`. Commands should stop when the context is done; running steps are cleared and point to the log.
- A second signal, or the grace period running out (10 seconds by default), exits with code `130`.
- Commands should expect to start in a state where cleanup never ran, and avoid relying on cleanup at all where possible.

### Configuration

With `command.WithConfig()`, flag defaults are read in this order, each source overriding the last:

1. `/etc/<app>/config.yaml` (or `.json`)
2. `~/.config/<app>/config.yaml` (or `.json`), following `XDG_CONFIG_HOME`
3. the closest `.<app>.yaml` in the current directory or a parent directory
4. environment variables bound with `Meta.EnvFlag`
5. flags passed on the command line

Top-level keys set framework and app-global flags, and maps keyed by a command path set the flags of that command:

```yaml
quiet: true
lollipop-eat:
  count: 2
```

The built-in `config` command shows the effective values and where each one comes from, and `--debug` lists the source of every flag value. `ConfigDir`, `CacheDir`, `DataDir`, `StateDir` and `RuntimeDir` return the XDG directories of the app.

### Environment variables

| Variable | Effect |
| --- | --- |
| `NO_COLOR`, `<APP>_NO_COLOR` | Disable color |
| `FORCE_COLOR` | Enable color without checking for a terminal |
| `TERM=dumb` | Disable color and animations |
| `DEBUG`, `<APP>_DEBUG` | Enable debugging output, like `--debug` |
| `<APP>_VERBOSE` | Stream every line of step output |
| `<APP>_DEBUG_LOG` | Write debugging output and step logs to this file |
| `<APP>_PAGER`, `PAGER`, `LESS` | Choose and configure the pager |
| `<APP>_<FLAG>` | Set flags bound with `Meta.EnvFlag`, such as `HELLO_WORLD_COUNT` |
| `VISUAL`, `EDITOR` | The editor used by `Meta.Editor` |
| `SHELL` | The shell used by `Meta.RunShell` and `completion` |
| `COLUMNS`, `LINES` | The terminal size used to wrap help and tables |
| `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY` | Proxies used by `Meta.HTTPClient` |
| `TMPDIR` | Where `Meta.TempFile` creates files |
| `XDG_*` | Where config, cache, data, state and runtime files live |

`<APP>` is the app name in uppercase, with other characters replaced by underscores, so `hello-world` uses `HELLO_WORLD_`.

### Naming

Pick a short, lowercase name that is easy to type, uses dashes only between words, and does not collide with a common command. `commandtest.ValidateCommands` checks this.

### Distribution

Go builds a single binary for each platform. The [`hello-world`](examples/hello-world) example includes a GoReleaser config, along with install and uninstall instructions.

`app completion --install` writes a completion script to the completion directory of bash, zsh or fish after asking for confirmation, and never changes shell startup files such as `~/.bashrc`. `app completion bash` prints the script instead.

### Analytics

`cli-skeleton` never sends usage or crash data anywhere. If your app collects any, ask for consent first, say what is collected and why, and make it easy to turn off.

## Built-in commands

| Command | Purpose |
| --- | --- |
| `help` | Show help for the app or a command |
| `examples` | Show examples for every command |
| `completion` | Print or install the shell completion script |
| `config` | Show the effective configuration, added by `WithConfig` |
| `docs` | Generate markdown or man page docs (hidden) |

A built-in command is only added when the app does not define a command with the same name.

## Testing

`commandtest.ValidateCommands` checks an app against the guidelines from a regular Go test: naming, long flag names, reserved short flags, consistent flag types, confusingly similar command names, the number of examples and positional arguments, and secrets passed through flags or environment variables.

```go
func TestCommands(t *testing.T) {
  commandtest.ValidateCommands(t, AppName, Commands)
}
```

To test this project, run the unit tests of the library and the examples, then the [bats](https://github.com/bats-core/bats-core) integration tests, which build the examples first:

```shell
make test
make test-integration
```

The scaffolding tests compare the generated code with golden files in `internal/scaffold/testdata/golden`. After changing the templates or an example spec, refresh them with `go test ./internal/scaffold -update`.
