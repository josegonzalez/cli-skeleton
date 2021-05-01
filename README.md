# cli-skeleton

An opinionated framework for building golang cli tools on top of [`mitchellh/cli`](https://github.com/mitchellh/cli).

## Why

While [`mitchellh/cli`](https://github.com/mitchellh/cli) gives quite a bit of code to allow folks to build cli tools on top of it, it does not provide enough structure to allow folks to get started quickly. This project aims to fill that void by implementing a skeleton based upon those provided by the hashicorp suite of tools.

See [Command Line Interface Guidelines](https://clig.dev/) for further reading on how to structure command line tools.


### Examples

For examples on how to perform various tasks, see the following examples:

- [`global`](examples/global): Shows how to implement "global" flags.
- [`hello-world`](examples/hello-world): The hello-world example.
- [`nil`](examples/nil): An example cli tool that does nothing. Useful for copying the `nil.go` command as a command template for your own cli tools.
