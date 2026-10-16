package platform

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"
)

// Capability states reported by Diagnose.
const (
	StateOK          = "ok"
	StateMissing     = "missing"
	StateUnknown     = "unknown"
	StateUnsupported = "unsupported"
)

// CommandTimeout bounds each probe so a hung helper cannot hang doctor.
const CommandTimeout = 10 * time.Second

// CommandRunner executes a fixed binary with arguments and returns its
// combined output. Runners must not invoke a shell. A missing executable is
// reported with an error wrapping exec.ErrNotFound (as *exec.Error does),
// which Diagnose maps to the missing state; any other failure maps to the
// unknown state.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
}

// OSRunner is the production CommandRunner.
type OSRunner struct{}

// Run locates the binary and executes it with a timeout. No shell is used.
func (OSRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	if _, err := exec.LookPath(name); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, CommandTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return string(out), err
}

// Check is one read-only capability result.
type Check struct {
	Name   string
	State  string
	Detail string
}

// Report is the full diagnosis.
type Report struct {
	Host   Info
	Checks []Check
}

// Diagnose probes the host and well-known tools. It never installs software,
// writes configuration, or requires elevation.
func Diagnose(ctx context.Context, host Info, run CommandRunner) Report {
	r := Report{Host: host}
	if host.Supported() {
		r.Checks = append(r.Checks, Check{Name: "Platform", State: StateOK, Detail: host.String() + " is supported"})
	} else {
		r.Checks = append(r.Checks, Check{Name: "Platform", State: StateUnsupported, Detail: host.String() + " is not supported in v1"})
	}
	r.Checks = append(r.Checks, probeTool(ctx, run, "WinGet", "winget", "--version"))
	r.Checks = append(r.Checks, probeTool(ctx, run, "mise", "mise", "--version"))
	r.Checks = append(r.Checks, probeTool(ctx, run, "Git", "git", "--version"))
	return r
}

func probeTool(ctx context.Context, run CommandRunner, label, name, arg string) Check {
	out, err := run.Run(ctx, name, arg)
	if err != nil {
		var execErr *exec.Error
		if errors.Is(err, exec.ErrNotFound) || errors.As(err, &execErr) {
			return Check{Name: label, State: StateMissing, Detail: name + " executable not found"}
		}
		if firstLine(out) != "" {
			return Check{Name: label, State: StateUnknown, Detail: name + " query failed: " + firstLine(out)}
		}
		return Check{Name: label, State: StateUnknown, Detail: name + " query failed: " + firstLine(err.Error())}
	}
	if v := firstLine(out); v != "" {
		return Check{Name: label, State: StateOK, Detail: v}
	}
	return Check{Name: label, State: StateUnknown, Detail: name + " returned no version output"}
}

// firstLine trims output to a bounded single line for display.
func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSpace(s)
	if len(s) > 128 {
		s = s[:128]
	}
	return s
}
