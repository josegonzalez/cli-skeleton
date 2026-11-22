package command

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

// WithConfig loads flag defaults from config files, applied in this order
// with later sources overriding earlier ones:
//
//  1. the system config, `/etc/<app>/config.yaml` (or `.json`)
//  2. the user config, `ConfigDir()/config.yaml` (or `.json`)
//  3. the project config, `.<app>.yaml`, found in the current directory or
//     the closest parent directory
//  4. environment variables bound with Meta.EnvFlag
//  5. flags passed on the command line
//
// Top-level keys set framework and app-global flags. Keys holding a map set
// the flags of a single command, named by its command path:
//
//	quiet: true
//	lollipop-eat:
//	  count: 2
//
// It also adds a `config` command that shows the effective configuration
// and where each value comes from.
func WithConfig() Option {
	return func(o *mainOptions) {
		o.config = true
	}
}

// configValue is a flag value read from a config file.
type configValue struct {
	value  string
	source string
}

// configFile is a config file location.
type configFile struct {
	path  string
	found bool
}

// configData holds the merged contents of the config files.
type configData struct {
	files    []configFile
	global   map[string]configValue
	commands map[string]map[string]configValue
}

// configPaths returns the config file candidates, lowest precedence first.
// The project file is the closest `.<app>.yaml` at or above the current
// directory.
func configPaths(appName string) []string {
	var paths []string

	system := filepath.Join("/etc", appName)
	if runtime.GOOS == "windows" {
		system = filepath.Join(os.Getenv("ProgramData"), appName)
	}
	paths = append(paths, filepath.Join(system, "config.yaml"), filepath.Join(system, "config.json"))

	if dir, err := ConfigDir(appName); err == nil {
		paths = append(paths, filepath.Join(dir, "config.yaml"), filepath.Join(dir, "config.json"))
	}

	if project := findProjectConfig(appName); project != "" {
		paths = append(paths, project)
	} else if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, "."+appName+".yaml"))
	}
	return paths
}

// findProjectConfig returns the closest `.<app>.yaml` at or above the
// current directory.
func findProjectConfig(appName string) string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		path := filepath.Join(dir, "."+appName+".yaml")
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// loadConfig reads and merges the config files of the app.
func loadConfig(appName string) (*configData, error) {
	cfg := &configData{
		global:   map[string]configValue{},
		commands: map[string]map[string]configValue{},
	}

	for _, path := range configPaths(appName) {
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			cfg.files = append(cfg.files, configFile{path: path})
			continue
		}
		if err != nil {
			return nil, &UserError{Message: fmt.Sprintf("Unable to read config file %s.", path), Err: err}
		}
		cfg.files = append(cfg.files, configFile{path: path, found: true})

		var values map[string]interface{}
		if err := yaml.Unmarshal(data, &values); err != nil {
			return nil, &UserError{
				Message: fmt.Sprintf("Invalid config file %s: %v", path, err),
				Hint:    "Fix the file, or move it aside to use the defaults.",
			}
		}

		for key, value := range values {
			if section, ok := value.(map[string]interface{}); ok {
				name := commandPath(key)
				if cfg.commands[name] == nil {
					cfg.commands[name] = map[string]configValue{}
				}
				for flagName, flagValue := range section {
					cfg.commands[name][flagName] = configValue{value: configString(flagValue), source: path}
				}
				continue
			}
			cfg.global[key] = configValue{value: configString(value), source: path}
		}
	}

	return cfg, nil
}

// configString converts a config value to the string form flags accept.
// Lists become comma-separated values.
func configString(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case []interface{}:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			parts = append(parts, fmt.Sprint(item))
		}
		return strings.Join(parts, ",")
	default:
		return fmt.Sprint(v)
	}
}

// lookup returns the config value for a flag of a command. Command
// sections take precedence over top-level keys.
func (c *configData) lookup(command string, name string) (configValue, bool) {
	if c == nil {
		return configValue{}, false
	}
	if section, ok := c.commands[commandPath(command)]; ok {
		if value, ok := section[name]; ok {
			return value, true
		}
	}
	value, ok := c.global[name]
	return value, ok
}

// applySettingsConfig applies the framework flags set in the config to s.
func (c *configData) applySettingsConfig(s *Settings) {
	if c == nil {
		return
	}
	for name, target := range settingFlags(s) {
		if value, ok := c.global[name]; ok {
			if b, err := parseBoolValue(value.value); err == nil {
				*target = b
			}
		}
	}
}

// parseBoolValue parses a boolean config value.
func parseBoolValue(value string) (bool, error) {
	f := flag.NewFlagSet("", flag.ContinueOnError)
	b := f.Bool("value", false, "")
	if err := f.Set("value", value); err != nil {
		return false, err
	}
	return *b, nil
}

// applyDefaults sets flag values from the config files and the environment,
// before the command line is parsed. Values are set without marking flags as
// changed, so flags passed on the command line still override them.
func (m *Meta) applyDefaults(flags *flag.FlagSet, command string) error {
	m.valueSources = map[string]string{}
	frameworkFlags := settingFlags(&Settings{})

	var err error
	flags.VisitAll(func(f *flag.Flag) {
		if err != nil {
			return
		}

		// Framework flags are resolved with the settings, before the UI is
		// built, from the config, the environment and the arguments.
		if _, ok := frameworkFlags[f.Name]; ok {
			return
		}

		if value, ok := m.config.lookup(command, f.Name); ok {
			if setErr := f.Value.Set(value.value); setErr != nil {
				err = &UsageError{
					Err:  fmt.Errorf("invalid value %q for %s in %s: %v", value.value, f.Name, value.source, setErr),
					Hint: "Fix the value in " + value.source + ".",
				}
				return
			}
			m.valueSources[f.Name] = value.source
		}

		if name, ok := m.envFlags[f.Name]; ok {
			if value, set := os.LookupEnv(name); set {
				if setErr := f.Value.Set(value); setErr != nil {
					err = &UsageError{
						Err:  fmt.Errorf("invalid value %q for %s in %s: %v", value, f.Name, name, setErr),
						Hint: "Fix or unset " + name + ".",
					}
					return
				}
				m.valueSources[f.Name] = "env " + name
			}
		}
	})
	return err
}

// recordFlagSources notes which flags were passed on the command line, and
// lists the source of each configured value under --debug.
func (m *Meta) recordFlagSources(flags *flag.FlagSet) {
	flags.Visit(func(f *flag.Flag) {
		m.valueSources[f.Name] = "flag"
	})

	if len(m.valueSources) == 0 {
		return
	}

	names := make([]string, 0, len(m.valueSources))
	for name := range m.valueSources {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if f := flags.Lookup(name); f != nil {
			m.Debugf("--%s=%s (from %s)", name, f.Value.String(), m.valueSources[name])
		}
	}
}

// ValueSource returns where the value of a flag came from: `flag`, `env
// <NAME>`, the path of a config file, or `default`.
func (m *Meta) ValueSource(name string) string {
	if source, ok := m.valueSources[name]; ok {
		return source
	}
	return "default"
}
