// Package version provides immutable compile-time build metadata.
//
// The Version, Commit, and Date variables are overridden at link time via
// -ldflags (e.g. -X hellokit/internal/version.Version=...). Defaults identify
// a local development build.
package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the semantic version of the binary.
	Version = "0.1.0-dev"
	// Commit is the VCS revision embedded at build time.
	Commit = "none"
	// Date is the build timestamp embedded at build time.
	Date = "unknown"
)

// Info describes the binary version and platform it was built for.
type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
}

// Get resolves the current build metadata alongside the runtime platform.
func Get() Info {
	return Info{
		Version: Version,
		Commit:  Commit,
		Date:    Date,
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	}
}

// String formats the version in human-readable form.
func (i Info) String() string {
	return fmt.Sprintf("hellokit %s (%s/%s) commit: %s date: %s",
		i.Version, i.OS, i.Arch, i.Commit, i.Date)
}
