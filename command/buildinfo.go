package command

import (
	"runtime"
	"runtime/debug"
)

// BuildInfo describes how the cli tool was built.
type BuildInfo struct {
	Version   string `json:"version"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	Commit    string `json:"commit,omitempty"`
	BuildDate string `json:"build_date,omitempty"`
	Modified  bool   `json:"modified,omitempty"`
}

// BuildInfo returns the version of the cli tool along with the Go version,
// platform and the version control details recorded by the Go toolchain.
func (m *Meta) BuildInfo() BuildInfo {
	info := BuildInfo{
		Version:   m.version,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}

	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return info
	}

	for _, setting := range bi.Settings {
		switch setting.Key {
		case "vcs.revision":
			info.Commit = setting.Value
		case "vcs.time":
			info.BuildDate = setting.Value
		case "vcs.modified":
			info.Modified = setting.Value == "true"
		}
	}

	if info.Version == "" && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		info.Version = bi.Main.Version
	}

	return info
}
