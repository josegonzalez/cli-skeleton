package command

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestXDGDirsFromEnv(t *testing.T) {
	base := t.TempDir()
	tests := []struct {
		env string
		fn  func(string) (string, error)
	}{
		{env: "XDG_CONFIG_HOME", fn: ConfigDir},
		{env: "XDG_CACHE_HOME", fn: CacheDir},
		{env: "XDG_DATA_HOME", fn: DataDir},
		{env: "XDG_STATE_HOME", fn: StateDir},
		{env: "XDG_RUNTIME_DIR", fn: RuntimeDir},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			t.Setenv(tt.env, base)
			got, err := tt.fn("hello-world")
			if err != nil {
				t.Fatal(err)
			}
			if want := filepath.Join(base, "hello-world"); got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	}
}

func TestXDGDirsFallBackToHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows uses the user config and cache dirs")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)

	tests := []struct {
		env  string
		fn   func(string) (string, error)
		want string
	}{
		{env: "XDG_CONFIG_HOME", fn: ConfigDir, want: filepath.Join(home, ".config", "app")},
		{env: "XDG_CACHE_HOME", fn: CacheDir, want: filepath.Join(home, ".cache", "app")},
		{env: "XDG_DATA_HOME", fn: DataDir, want: filepath.Join(home, ".local", "share", "app")},
		{env: "XDG_STATE_HOME", fn: StateDir, want: filepath.Join(home, ".local", "state", "app")},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			t.Setenv(tt.env, "")
			got, err := tt.fn("app")
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}

			t.Setenv(tt.env, "relative/path")
			got, err = tt.fn("app")
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("relative %s should be ignored: got %q", tt.env, got)
			}
		})
	}
}

func TestRuntimeDirUnset(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "")
	if _, err := RuntimeDir("app"); !errors.Is(err, ErrNoRuntimeDir) {
		t.Fatalf("expected ErrNoRuntimeDir, got %v", err)
	}
}

func TestTempFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)

	m := &Meta{appName: "hello-world"}
	f, err := m.TempFile("*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	if runtime.GOOS != "windows" && filepath.Dir(f.Name()) != dir {
		t.Errorf("temp file %q not created in TMPDIR %q", f.Name(), dir)
	}
	if !strings.HasPrefix(filepath.Base(f.Name()), "hello-world-") {
		t.Errorf("temp file %q should start with the app name", f.Name())
	}
}
