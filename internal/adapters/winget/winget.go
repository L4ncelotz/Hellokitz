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

// WinGet HRESULT error codes from AppInstallerErrors.h.
const (
	// ErrNoApplicationsFoundHResult is APPINSTALLER_CLI_ERROR_NO_APPLICATIONS_FOUND (0x8A150014).
	// In signed 32-bit representation it is -1978335212; in unsigned 32-bit it is 2316632084.
	ErrNoApplicationsFoundHResult = 0x8A150014
	// ErrNoApplicationsFoundLegacy is the 8-bit truncated exit code (20 / 0x14)
	// emitted when executed through POSIX wrappers or cmd subshells.
	ErrNoApplicationsFoundLegacy = 20
)

func isNoMatchExitCode(code int) bool {
	return uint32(code) == ErrNoApplicationsFoundHResult || code == ErrNoApplicationsFoundLegacy
}

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
func (a Adapter) LookupInstalled(ctx context.Context, id string) plan.Observation {
	if id == "" {
		return plan.Observation{State: plan.StateUnknown, Detail: "empty package id"}
	}
	out, err := a.Run.Run(ctx, Binary,
		"list", "--exact", "--id", id, "--source", "winget", "--disable-interactivity")
	if err == nil {
		if version, ok := exactMatch(out, id); ok {
			if version != "" {
				return plan.Observation{
					State:   plan.StateSatisfied,
					Version: version,
					Detail:  fmt.Sprintf("installed version %s (exact ID match)", version),
				}
			}
			return plan.Observation{
				State:  plan.StateSatisfied,
				Detail: "installed (exact ID match)",
			}
		}
		return plan.Observation{
			State:  plan.StateUnknown,
			Detail: fmt.Sprintf("winget list returned no exact match for %q; check ID casing", id),
		}
	}
	if isNoMatchExitCode(exitCode(err)) && strings.Contains(out, NoMatchMessage) {
		return plan.Observation{
			State:  plan.StateMissing,
			Detail: fmt.Sprintf("no installed package from source winget with exact ID %q", id),
		}
	}
	if errors.Is(err, exec.ErrNotFound) {
		return plan.Observation{
			State:  plan.StateUnknown,
			Detail: "cannot determine installation state: winget executable not found",
		}
	}
	if firstLine(out) != "" {
		return plan.Observation{
			State:  plan.StateUnknown,
			Detail: fmt.Sprintf("winget list failed: %s", firstLine(out)),
		}
	}
	return plan.Observation{
		State:  plan.StateUnknown,
		Detail: fmt.Sprintf("winget list failed: %s", firstLine(err.Error())),
	}
}

// LookupApplication implements plan.Inventory for the winget source. Other
// sources are unknown to this adapter, never silently mapped.
func (a Adapter) LookupApplication(id, source string) plan.Observation {
	if source != "winget" {
		return plan.Observation{
			State:  plan.StateUnknown,
			Detail: fmt.Sprintf("source %q is not handled by the WinGet adapter", source),
		}
	}
	// plan.Inventory carries no context; the runner still bounds each probe.
	return a.LookupInstalled(context.Background(), id)
}

// colSpan tracks a column header's name and byte offset in the header line.
type colSpan struct {
	name  string
	start int
}

// exactMatch scans WinGet table output for a row whose Id column exactly
// matches id (case-sensitive, mirroring --exact). It anchors column offsets
// to the table header and separator line to avoid false matches when another
// package's name matches id. It returns the installed version (empty if
// missing or reported as unknown) and true on exact match.
func exactMatch(out, id string) (string, bool) {
	lines := strings.Split(out, "\n")
	headerIdx := -1
	for i := range len(lines) - 1 {
		header := strings.TrimRight(lines[i], "\r\n")
		sep := strings.TrimRight(lines[i+1], "\r\n")
		if isSeparator(sep) {
			words := strings.Fields(header)
			hasName := false
			hasID := false
			for _, w := range words {
				if w == "Name" {
					hasName = true
				}
				if w == "Id" || w == "ID" {
					hasID = true
				}
			}
			if hasName && hasID {
				headerIdx = i
				break
			}
		}
	}
	if headerIdx < 0 {
		return "", false
	}

	header := strings.TrimRight(lines[headerIdx], "\r\n")
	var cols []colSpan
	inWord := false
	start := 0
	for idx, r := range header {
		if r != ' ' && r != '\t' {
			if !inWord {
				start = idx
				inWord = true
			}
		} else {
			if inWord {
				cols = append(cols, colSpan{name: header[start:idx], start: start})
				inWord = false
			}
		}
	}
	if inWord {
		cols = append(cols, colSpan{name: header[start:], start: start})
	}

	idColIdx := -1
	verColIdx := -1
	for idx, c := range cols {
		if c.name == "Id" || c.name == "ID" {
			idColIdx = idx
		} else if c.name == "Version" {
			verColIdx = idx
		}
	}
	if idColIdx < 0 {
		return "", false
	}

	idStart := cols[idColIdx].start
	idEnd := len(header)
	if idColIdx+1 < len(cols) {
		idEnd = cols[idColIdx+1].start
	}

	verStart := -1
	verEnd := -1
	if verColIdx >= 0 {
		verStart = cols[verColIdx].start
		verEnd = len(header)
		if verColIdx+1 < len(cols) {
			verEnd = cols[verColIdx+1].start
		}
	}

	for _, line := range lines[headerIdx+2:] {
		line = strings.TrimRight(line, "\r\n")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || isSeparator(trimmed) {
			continue
		}
		if len(line) < idStart {
			continue
		}
		boundIDEnd := len(line)
		if idEnd < boundIDEnd {
			boundIDEnd = idEnd
		}
		rowID := strings.TrimSpace(line[idStart:boundIDEnd])
		if rowID == id {
			version := ""
			if verStart >= 0 && len(line) >= verStart {
				boundVerEnd := len(line)
				if verEnd >= 0 && verEnd < boundVerEnd {
					boundVerEnd = verEnd
				}
				v := strings.TrimSpace(line[verStart:boundVerEnd])
				if !strings.EqualFold(v, "unknown") {
					version = v
				}
			}
			return version, true
		}
	}
	return "", false
}

func isSeparator(line string) bool {
	hasDash := false
	for _, r := range line {
		if r == '-' {
			hasDash = true
		} else if r != ' ' && r != '\t' {
			return false
		}
	}
	return hasDash
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
