package platform

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestHost_MatchesRuntime(t *testing.T) {
	h := Host()
	if h.OS != runtime.GOOS || h.Arch != runtime.GOARCH {
		t.Errorf("Host() = %v, want %s/%s", h, runtime.GOOS, runtime.GOARCH)
	}
}

func TestInfo_Supported(t *testing.T) {
	cases := []struct {
		host Info
		want bool
	}{
		{Info{"windows", "amd64"}, true},
		{Info{"windows", "arm64"}, false},
		{Info{"linux", "amd64"}, false},
		{Info{"darwin", "arm64"}, false},
		{Info{"", ""}, false},
	}
	for _, tc := range cases {
		if got := tc.host.Supported(); got != tc.want {
			t.Errorf("(%v).Supported() = %v, want %v", tc.host, got, tc.want)
		}
	}
}

// fakeRunner is a deterministic CommandRunner for tests.
type fakeRunner struct {
	fn func(name string, args ...string) (string, error)
}

func (f fakeRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	return f.fn(name, args...)
}

func okRunner() fakeRunner {
	return fakeRunner{fn: func(name string, _ ...string) (string, error) {
		switch name {
		case "winget":
			return "v1.10.340\n", nil
		case "mise":
			return "mise 2026.1.0 windows-x64\n", nil
		case "git":
			return "git version 2.54.0.windows.1\n", nil
		}
		return "", &exec.Error{Name: name, Err: exec.ErrNotFound}
	}}
}

func TestDiagnose_AllOK(t *testing.T) {
	r := Diagnose(context.Background(), Info{"windows", "amd64"}, okRunner())
	if len(r.Checks) != 4 {
		t.Fatalf("expected 4 checks, got %+v", r.Checks)
	}
	for _, c := range r.Checks {
		if c.State != StateOK {
			t.Errorf("check %q = %s (%s), want ok", c.Name, c.State, c.Detail)
		}
	}
}

func TestDiagnose_UnsupportedHost(t *testing.T) {
	r := Diagnose(context.Background(), Info{"linux", "arm64"}, okRunner())
	if r.Checks[0].Name != "Platform" || r.Checks[0].State != StateUnsupported {
		t.Errorf("first check should be unsupported platform, got %+v", r.Checks[0])
	}
	// Tool probes still run: diagnosis stays read-only on any host.
	for _, c := range r.Checks[1:] {
		if c.State != StateOK {
			t.Errorf("tool check %q = %s, want ok", c.Name, c.State)
		}
	}
}

func TestDiagnose_MissingAndFailures(t *testing.T) {
	run := fakeRunner{fn: func(name string, _ ...string) (string, error) {
		switch name {
		case "winget":
			return "", &exec.Error{Name: name, Err: exec.ErrNotFound}
		case "mise":
			return "partial output", errors.New("exit status 1")
		case "git":
			return "", nil
		}
		return "", errors.New("unexpected probe " + name)
	}}
	r := Diagnose(context.Background(), Info{"windows", "amd64"}, run)
	states := map[string]string{}
	for _, c := range r.Checks {
		states[c.Name] = c.State
	}
	if states["WinGet"] != StateMissing {
		t.Errorf("WinGet = %s, want missing", states["WinGet"])
	}
	if states["mise"] != StateUnknown || !strings.Contains(reportDetail(r, "mise"), "partial output") {
		t.Errorf("mise should be unknown with probe output, got %+v", r.Checks)
	}
	if states["Git"] != StateUnknown {
		t.Errorf("Git = %s, want unknown on empty output", states["Git"])
	}
}

func reportDetail(r Report, name string) string {
	for _, c := range r.Checks {
		if c.Name == name {
			return c.Detail
		}
	}
	return ""
}

func TestDiagnose_TimeoutIsUnknown(t *testing.T) {
	run := fakeRunner{fn: func(_ string, _ ...string) (string, error) {
		return "", context.DeadlineExceeded
	}}
	r := Diagnose(context.Background(), Info{"windows", "amd64"}, run)
	for _, c := range r.Checks[1:] {
		if c.State != StateUnknown {
			t.Errorf("tool check %q = %s, want unknown on timeout", c.Name, c.State)
		}
	}
}
