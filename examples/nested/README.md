# nested

An example cli tool for the `cli-skeleton` project with nested commands, such as `nested lollipop eat`. It shows most of the helpers that make a cli tool follow the [Command Line Interface Guidelines](https://clig.dev/).

## Building

```shell
# substitute the version number as desired
go build -ldflags "-X main.Version=0.1.0"
```

## Usage

```
An example cli tool with nested commands for managing lollipops.

USAGE
  nested <command> [flags]

EXAMPLES
  List the lollipops
    $ nested lollipop list

  Eat a cherry lollipop
    $ nested lollipop eat cherry

COMMANDS
  completion   Print or install the shell completion script
  config       Show the effective configuration and where it comes from
  examples     Show examples for every command
  lollipop     Commands for lollipop
  version      Return the version of the binary
```

## What it shows

### Nested commands

Commands are registered under their full name, such as `lollipop eat`. `cli-skeleton` adds the `lollipop` parent command, which lists its subcommands when run on its own or with `--help`.

```go
func Commands(ctx context.Context, meta command.Meta) map[string]cli.CommandFactory {
  return map[string]cli.CommandFactory{
    "lollipop eat": func() (cli.Command, error) {
      return &commands.EatCommand{Meta: meta}, nil
    },
    "lollipop list": func() (cli.Command, error) {
      return &commands.ListCommand{Meta: meta}, nil
    },
  }
}
```

The `lollipop ls` alias is registered with `command.WithAliases`, and `lollipop munch` is a deprecated command that warns before running and is hidden from the command list.

### Tables, `--plain` and `--json`

`lollipop list` uses `OutputTable`, which prints aligned columns for people, one tab-separated record per line under `--plain`, and an array of objects under `--json`.

```shell
nested lollipop list
nested lollipop list --plain | cut -f1
nested lollipop list --json --all
```

### Confirmation, prompts and validation

`lollipop eat` asks before eating, unless `--force` is passed, and shows what would happen with `--dry-run`. When the flavor is missing and stdin is a terminal, it is prompted for; otherwise the command prints its concise help. Flavors are validated before the command runs, and `--count` can also be set with `NESTED_COUNT` or in a config file.

```shell
nested lollipop eat cherry
nested lollipop eat grape --count 2 --force
NESTED_COUNT=3 nested lollipop eat lemon --dry-run
```

### Progress, locks and interrupts

`lollipop savor` runs numbered steps, an external command and a progress bar. On a terminal the running step shows a spinner and its recent output; elsewhere a line is printed as each step starts and ends. `--drop` makes the last step fail, which shows the output of the step and the path of the full log.

Only one lollipop can be savored at a time: a second run fails right away, or waits with `--wait`. Pressing Ctrl+C stops the command, and pressing it again forces it to exit.

```shell
nested lollipop savor --duration 10s
nested lollipop savor --drop
nested lollipop savor --wait
```

### Configuration

`command.WithConfig()` reads flag defaults from `/etc/nested/config.yaml`, `~/.config/nested/config.yaml` and the closest `.nested.yaml`. `nested config` shows the effective values and where they come from.

```yaml
# ~/.config/nested/config.yaml
quiet: true
lollipop-eat:
  count: 2
```
