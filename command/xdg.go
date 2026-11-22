package command

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

// ErrNoRuntimeDir is returned by RuntimeDir when XDG_RUNTIME_DIR is not set.
var ErrNoRuntimeDir = errors.New("XDG_RUNTIME_DIR is not set")

// ConfigDir returns the directory for the app's user configuration:
// `$XDG_CONFIG_HOME/<app>`, falling back to `~/.config/<app>`. On Windows
// it falls back to the user config dir instead.
func ConfigDir(appName string) (string, error) {
	return xdgDir("XDG_CONFIG_HOME", appName, filepath.Join(".config"), os.UserConfigDir)
}

// CacheDir returns the directory for the app's cached data:
// `$XDG_CACHE_HOME/<app>`, falling back to `~/.cache/<app>`. On Windows it
// falls back to the user cache dir instead.
func CacheDir(appName string) (string, error) {
	return xdgDir("XDG_CACHE_HOME", appName, filepath.Join(".cache"), os.UserCacheDir)
}

// DataDir returns the directory for the app's data files:
// `$XDG_DATA_HOME/<app>`, falling back to `~/.local/share/<app>`. On
// Windows it falls back to the user config dir instead.
func DataDir(appName string) (string, error) {
	return xdgDir("XDG_DATA_HOME", appName, filepath.Join(".local", "share"), os.UserConfigDir)
}

// StateDir returns the directory for the app's state, such as logs:
// `$XDG_STATE_HOME/<app>`, falling back to `~/.local/state/<app>`. On
// Windows it falls back to the user cache dir instead.
func StateDir(appName string) (string, error) {
	return xdgDir("XDG_STATE_HOME", appName, filepath.Join(".local", "state"), os.UserCacheDir)
}

// RuntimeDir returns the directory for the app's runtime files, such as
// locks: `$XDG_RUNTIME_DIR/<app>`. It returns ErrNoRuntimeDir when
// XDG_RUNTIME_DIR is not set, since the spec has no fallback.
func RuntimeDir(appName string) (string, error) {
	base := os.Getenv("XDG_RUNTIME_DIR")
	if base == "" || !filepath.IsAbs(base) {
		return "", ErrNoRuntimeDir
	}
	return filepath.Join(base, appName), nil
}

// xdgDir resolves an XDG base directory for the app. Relative values of the
// env var are ignored, as the XDG spec requires.
func xdgDir(env string, appName string, homeFallback string, windowsFallback func() (string, error)) (string, error) {
	if base := os.Getenv(env); base != "" && filepath.IsAbs(base) {
		return filepath.Join(base, appName), nil
	}

	if runtime.GOOS == "windows" {
		base, err := windowsFallback()
		if err != nil {
			return "", err
		}
		return filepath.Join(base, appName), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, homeFallback, appName), nil
}

// TempFile creates a new temporary file in the directory named by TMPDIR,
// or the system default when it is not set. The file name starts with the
// app name, and the caller is responsible for removing the file.
func (m *Meta) TempFile(pattern string) (*os.File, error) {
	prefix := m.appName
	if prefix == "" {
		prefix = "cli"
	}
	return os.CreateTemp("", prefix+"-"+pattern)
}
