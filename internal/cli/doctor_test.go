package cli

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/L4ncelotz/Hellokitz/internal/platform"
)

type fakeDoctorRunner struct {
	fn func(name string, args ...string) (string, error)
}

func (f fakeDoctorRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	return f.fn(name, args...)
}

func withDoctorRunner(t *testing.T, r platform.CommandRunner) {
	t.Helper()
	old := doctorRunner
	doctorRunner = r
	t.Cleanup(func() { doctorRunner = old })
}

func TestDoctorCmd_AllOK(t *testing.T) {
	withDoctorRunner(t, fakeDoctorRunner{fn: func(name string, _ ...string) (string, error) {
		return name + " 1.0-test", nil
	}})

	out, err := runArgs(t, "doctor")
	if err != nil {
		t.Fatalf("Execute(doctor) returned error: %v", err)
	}
	for _, want := range []string{"Host: " + runtime.GOOS + "/" + runtime.GOARCH, "[OK] Platform", "[OK] WinGet", "1.0-test"} {
		if !strings.Contains(out, want) {
			t.Errorf("doctor output missing %q, got:\n%s", want, out)
		}
	}
}

func TestDoctorCmd_MissingAndUnknown(t *testing.T) {
	withDoctorRunner(t, fakeDoctorRunner{fn: func(name string, _ ...string) (string, error) {
		if name == "winget" {
			return "", &exec.Error{Name: name, Err: exec.ErrNotFound}
		}
		return "", context.DeadlineExceeded
	}})

	out, err := runArgs(t, "doctor")
	if err != nil {
		t.Fatalf("Execute(doctor) returned error: %v", err)
	}
	for _, want := range []string{"[MISSING] WinGet", "[UNKNOWN] mise", "[UNKNOWN] Git"} {
		if !strings.Contains(out, want) {
			t.Errorf("doctor output missing %q, got:\n%s", want, out)
		}
	}
}

func TestDoctorCmd_RejectsArgs(t *testing.T) {
	withDoctorRunner(t, fakeDoctorRunner{fn: func(_ string, _ ...string) (string, error) {
		return "ok", nil
	}})

	if _, err := runArgs(t, "doctor", "extra"); err == nil {
		t.Fatal("Execute(doctor extra) expected error, got nil")
	}
}
