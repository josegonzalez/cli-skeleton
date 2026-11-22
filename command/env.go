package command

import (
	"strings"

	flag "github.com/spf13/pflag"
)

// FlagEnv returns the env var bound to a flag by EnvFlag, such as
// `HELLO_WORLD_COUNT` for the flag `count` of the app `hello-world`.
func FlagEnv(appName string, flagName string) string {
	return AppEnv(appName, EnvPrefix(strings.ReplaceAll(flagName, "-", "_")))
}

// EnvFlag lets the environment set the default of a flag already defined
// on fs, through `<APP>_<FLAG>`, such as `HELLO_WORLD_COUNT` for `--count`.
// The variable is shown in help. Flags passed on the command line still
// take precedence.
//
// Call it from a command's FlagSet method, after defining the flag. Do not
// bind secrets to environment variables: they leak into child processes and
// logs. Read secrets from a file instead, with ReadSecret.
func (m *Meta) EnvFlag(fs *flag.FlagSet, name string) {
	f := fs.Lookup(name)
	if f == nil {
		return
	}

	env := FlagEnv(m.appName, name)
	if env == "" {
		return
	}

	if m.envFlags == nil {
		m.envFlags = map[string]string{}
	}
	if !strings.Contains(f.Usage, "[env: ") {
		f.Usage += " [env: " + env + "]"
	}
	m.envFlags[name] = env
}
