// Package winget probes WinGet read-only state: availability and exact-ID
// installed lookup.
//
// It never installs, upgrades, accepts source agreements, or invokes a
// shell: every probe passes a fixed argument array (with interactive prompts
// disabled) to the winget binary. Anything unverifiable — timeouts,
// failures, ambiguous rows, or unrecognized locales — is reported as
// unknown, never assumed installed or missing. A package counts as
// installed only on an exact, case-sensitive identity match, mirroring
// `winget list --exact` semantics.
package winget

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"hellokit/internal/plan"
	"hellokit/internal/platform"
)

// Binary is the WinGet executable name.
const Binary = "winget"

// NoMatchExitCode is the process exit code observed when an exact list
// query matches nothing (WinGet v1.29.290, en-US). Other versions or locales
// may differ and then fail closed to unknown: confirmed absence additionally
// requires NoMatchMessage.
const NoMatchExitCode = 20

// NoMatchMessage is the English no-match message. Localized variants are
// intentionally not recognized: without the exact signal, absence cannot be
// confirmed and the state stays unknown.
const NoMatchMessage = "No installed package found matching input criteria."

// Availability describes WinGet readiness.
type Availability struct {
	Available bool
	Version   string
	Detail    string
}

// Adapter probes WinGet through an injectable runner.
type Adapter struct {
	Run platform.CommandRunner
}

// Compile-time guarantee: the adapter feeds plan.Build directly.
var _ plan.Inventory = Adapter{}

// ProbeAvailability reports whether WinGet answers a version query.
func (a Adapter) ProbeAvailability(ctx context.Context) Availability {
	out, err := a.Run.Run(ctx, Binary, "--disable-interactivity", "--version")
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return Availability{Detail: "winget executable not found"}
		}
		return Availability{Detail: "winget query failed: " + firstLine(err.Error())}
	}
	if v := firstLine(out); v != "" {
		return Availability{Available: true, Version: v}
	}
	return Availability{Detail: "winget returned no version output"}
}

// LookupInstalled inspects one exact package ID. Source scope is always the
// winget community repository; see LookupApplication for gating.
func (a Adapter) LookupInstalled(ctx context.Context, id string) (state, detail string) {
	if id == "" {
		return plan.StateUnknown, "empty package id"
	}
	out, err := a.Run.Run(ctx, Binary,
		"list", "--exact", "--id", id, "--source", "winget", "--disable-interactivity")
	if err == nil {
		if version, ok := exactMatch(out, id); ok {
			if version != "" {
				return plan.StateSatisfied, fmt.Sprintf("installed version %s (exact ID match)", version)
			}
			return plan.StateSatisfied, "installed (exact ID match)"
		}
		return plan.StateUnknown, fmt.Sprintf("winget list returned no exact match for %q; check ID casing", id)
	}
	if exitCode(err) == NoMatchExitCode && strings.Contains(out, NoMatchMessage) {
		return plan.StateMissing, fmt.Sprintf("no installed package from source winget with exact ID %q", id)
	}
	if errors.Is(err, exec.ErrNotFound) {
		return plan.StateUnknown, "cannot determine installation state: winget executable not found"
	}
	if firstLine(out) != "" {
		return plan.StateUnknown, fmt.Sprintf("winget list failed: %s", firstLine(out))
	}
	return plan.StateUnknown, fmt.Sprintf("winget list failed: %s", firstLine(err.Error()))
}

// LookupApplication implements plan.Inventory for the winget source. Other
// sources are unknown to this adapter, never silently mapped.
func (a Adapter) LookupApplication(id, source string) (string, string) {
	if source != "winget" {
		return plan.StateUnknown, fmt.Sprintf("source %q is not handled by the WinGet adapter", source)
	}
	// plan.Inventory carries no context; the runner still bounds each probe.
	return a.LookupInstalled(context.Background(), id)
}

// exactMatch scans WinGet table rows for a whitespace-delimited token equal
// to id (case-sensitive, mirroring --exact). The header line and separator
// lines are skipped. The reported version is the token following the ID
// column occurrence (the last exact token on the line, since the name
// column precedes it).
func exactMatch(out, id string) (string, bool) {
	header := true
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || isSeparator(line) {
			continue
		}
		if header {
			header = false
			continue
		}
		toks := strings.Fields(line)
		last := -1
		for i, tok := range toks {
			if tok == id {
				last = i
			}
		}
		if last < 0 {
			continue
		}
		if last+1 < len(toks) && !strings.EqualFold(toks[last+1], "unknown") {
			return toks[last+1], true
		}
		return "", true
	}
	return "", false
}

func isSeparator(line string) bool {
	for _, r := range line {
		if r != '-' && r != ' ' && r != '\t' {
			return false
		}
	}
	return len(line) > 0
}

// exitCode extracts a process exit code, or -1 when the error carries none
// (timeouts, transport failures). *exec.ExitError implements ExitCode.
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var coder interface{ ExitCode() int }
	if errors.As(err, &coder) {
		return coder.ExitCode()
	}
	return -1
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
