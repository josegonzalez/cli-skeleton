package command

import (
	"os"
	"strings"

	"golang.org/x/term"
)

const (
	// EnvForceColor is an env var that forces colored output, skipping
	// terminal detection. NO_COLOR still takes precedence.
	EnvForceColor = `FORCE_COLOR`
)

// IsTerminal reports whether f is an interactive terminal.
func IsTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// EnvPrefix returns the prefix used for app-specific environment variables.
// The app name is upper-cased, and any character that is not a letter or
// digit is replaced with an underscore, so `hello-world` becomes `HELLO_WORLD`.
func EnvPrefix(appName string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(appName) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}

	prefix := b.String()
	if prefix != "" && prefix[0] >= '0' && prefix[0] <= '9' {
		prefix = "_" + prefix
	}
	return prefix
}

// AppEnv returns the name of an app-specific environment variable, such as
// `HELLO_WORLD_NO_COLOR` for the app `hello-world` and the suffix `NO_COLOR`.
func AppEnv(appName string, suffix string) string {
	prefix := EnvPrefix(appName)
	if prefix == "" {
		return ""
	}
	return prefix + "_" + suffix
}

// ColorEnabled reports whether colored output should be written to f.
//
// Color is disabled when the --no-color flag is set, when NO_COLOR or
// <APP>_NO_COLOR is set to a non-empty value, or when TERM is `dumb`.
// FORCE_COLOR (set to anything other than `0` or `false`) enables color
// without checking whether f is a terminal. Otherwise color is only enabled
// when f is a terminal.
func ColorEnabled(f *os.File, appName string, noColorFlag bool) bool {
	return colorEnabled(IsTerminal(f), appName, noColorFlag)
}

func colorEnabled(isTerminal bool, appName string, noColorFlag bool) bool {
	if noColorFlag {
		return false
	}

	if os.Getenv(EnvCLINoColor) != "" {
		return false
	}

	if name := AppEnv(appName, EnvCLINoColor); name != "" && os.Getenv(name) != "" {
		return false
	}

	if forceColor := os.Getenv(EnvForceColor); forceColor != "" && forceColor != "0" && forceColor != "false" {
		return true
	}

	if os.Getenv("TERM") == "dumb" {
		return false
	}

	return isTerminal
}
