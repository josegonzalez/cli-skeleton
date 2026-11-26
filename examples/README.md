# examples

- [`global`](global): Shows how to implement "global" flags.
- [`hello-world`](hello-world): The hello-world example.
- [`human-readable-logging`](human-readable-logging): The hello-world example with nicer log output.
- [`nested`](nested): An example cli tool with nested commands, showing tables, confirmations, prompts, progress steps, locks, configuration and more.
- [`nil`](nil): An example cli tool that does nothing. Useful for copying the `nil.go` command as a command template for your own cli tools.
- [`zerolog-logging`](zerolog-logging): The hello-world example with zerolog logger support.

Each example has a `scaffold.yaml` spec. `cli-skeleton new <directory> --spec <example>/scaffold.yaml` generates the example, apart from the bodies of its `Run` methods.
