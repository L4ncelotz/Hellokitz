// Package platform describes the host and probes read-only capabilities.
//
// Nothing here mutates the machine: probes run fixed read-only queries
// (e.g. `winget --version`) through parameterized argument arrays, never a
// shell. Unverifiable state is reported as unknown, never assumed.
package platform

import "runtime"

// Info identifies the host OS and architecture.
type Info struct {
	OS   string
	Arch string
}

// Host returns the OS and architecture of the running process.
func Host() Info {
	return Info{OS: runtime.GOOS, Arch: runtime.GOARCH}
}

// Supported reports whether schema v1 profiles target this host.
func (i Info) Supported() bool {
	return i.OS == "windows" && i.Arch == "amd64"
}

func (i Info) String() string {
	return i.OS + "/" + i.Arch
}
