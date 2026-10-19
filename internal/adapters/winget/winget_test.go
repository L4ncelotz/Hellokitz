package winget

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/L4ncelotz/Hellokitz/internal/plan"
)

// fakeRunner records invocations and replays canned results.
type fakeRunner struct {
	calls [][2]string
	fn    func(name string, args []string) (string, error)
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	f.calls = append(f.calls, [2]string{name, strings.Join(args, "\x00")})
	return f.fn(name, args)
}

func (f *fakeRunner) argv(i int) []string {
	return strings.Split(f.calls[i][1], "\x00")
}

// fakeExitError carries a process exit code like *exec.ExitError does.
type fakeExitError struct {
	code int
	msg  string
}

func (e fakeExitError) Error() string { return e.msg }
func (e fakeExitError) ExitCode() int { return e.code }

const matchTable = "Name Id      Version Available Source\r\n" +
	"-------------------------------------\r\n" +
	"Git  Git.Git 2.54.0  2.55.0.5  winget\r\n"

func TestProbeAvailability(t *testing.T) {
	runner := &fakeRunner{fn: func(_ string, _ []string) (string, error) {
		return "v1.29.290\n", nil
	}}
	a := Adapter{Run: runner}
	av := a.ProbeAvailability(context.Background())
	if !av.Available || av.Version != "v1.29.290" {
		t.Errorf("expected available v1.29.290, got %+v", av)
	}
	if got := runner.argv(0); len(got) != 2 || got[0] != "--disable-interactivity" || got[1] != "--version" {
		t.Errorf("unexpected probe argv: %q", got)
	}
}

func TestProbeMissing(t *testing.T) {
	runner := &fakeRunner{fn: func(name string, _ []string) (string, error) {
		return "", &exec.Error{Name: name, Err: exec.ErrNotFound}
	}}
	av := Adapter{Run: runner}.ProbeAvailability(context.Background())
	if av.Available || !strings.Contains(av.Detail, "not found") {
		t.Errorf("expected missing winget, got %+v", av)
	}
}

func TestProbeFailureIsNotMissing(t *testing.T) {
	runner := &fakeRunner{fn: func(_ string, _ []string) (string, error) {
		return "", errors.New("boom")
	}}
	av := Adapter{Run: runner}.ProbeAvailability(context.Background())
	if av.Available || !strings.Contains(av.Detail, "failed") {
		t.Errorf("expected failed (not missing) probe, got %+v", av)
	}
}

func TestProbeEmptyOutput(t *testing.T) {
	runner := &fakeRunner{fn: func(_ string, _ []string) (string, error) {
		return "\n", nil
	}}
	av := Adapter{Run: runner}.ProbeAvailability(context.Background())
	if av.Available {
		t.Errorf("empty version output must not count as available: %+v", av)
	}
}

func matchRunner(t *testing.T, output string, err error) (*fakeRunner, Adapter) {
	t.Helper()
	runner := &fakeRunner{fn: func(_ string, _ []string) (string, error) {
		return output, err
	}}
	return runner, Adapter{Run: runner}
}

func TestLookupExactMatch(t *testing.T) {
	runner, a := matchRunner(t, matchTable, nil)
	obs := a.LookupInstalled(context.Background(), "Git.Git")
	if obs.State != plan.StateSatisfied {
		t.Fatalf("expected satisfied, got %s (%s)", obs.State, obs.Detail)
	}
	if obs.Version != "2.54.0" {
		t.Errorf("expected version 2.54.0, got %q", obs.Version)
	}
	if !strings.Contains(obs.Detail, "2.54.0") {
		t.Errorf("detail should carry detected version, got %q", obs.Detail)
	}
	want := []string{"list", "--exact", "--id", "Git.Git", "--source", "winget", "--disable-interactivity"}
	if got := runner.argv(0); !equal(got, want) {
		t.Errorf("argv = %q, want %q", got, want)
	}
	for _, tok := range runner.argv(0) {
		if tok == "cmd" || tok == "powershell" || tok == "/c" || tok == "-c" || tok == "sh" {
			t.Errorf("shell token %q in argv; argument-array execution required", tok)
		}
	}
}

func TestLookupConfirmedAbsence(t *testing.T) {
	_, a := matchRunner(t, "No installed package found matching input criteria.\n",
		fakeExitError{code: 20, msg: "exit status 20"})
	obs := a.LookupInstalled(context.Background(), "No.Such.App")
	if obs.State != plan.StateMissing {
		t.Errorf("expected missing, got %s (%s)", obs.State, obs.Detail)
	}
}

func TestLookupNoMatchExitCodes(t *testing.T) {
	codes := []struct {
		name string
		code int
	}{
		{"legacy 20", 20},
		{"unsigned HRESULT 0x8A150014", int(uint32(0x8A150014))},
		{"signed HRESULT -1978335212", -1978335212},
	}
	for _, tc := range codes {
		t.Run(tc.name, func(t *testing.T) {
			_, a := matchRunner(t, "No installed package found matching input criteria.\n",
				fakeExitError{code: tc.code, msg: fmt.Sprintf("exit status %d", tc.code)})
			obs := a.LookupInstalled(context.Background(), "No.Such.App")
			if obs.State != plan.StateMissing {
				t.Errorf("code %d: expected missing, got %s (%s)", tc.code, obs.State, obs.Detail)
			}
		})
	}
}

func TestLookupCaseSensitive(t *testing.T) {
	// --exact is case-sensitive: a lowercase query against a cased row is
	// not a confirmed match and must not be reported missing either.
	_, a := matchRunner(t, matchTable, nil)
	obs := a.LookupInstalled(context.Background(), "git.git")
	if obs.State != plan.StateUnknown {
		t.Errorf("case-variant id must be unknown, got %s", obs.State)
	}
}
func TestLookupSubstringIsNotMatch(t *testing.T) {
	table := "Name   Id                Version Source\n" +
		"---------------------------------------\n" +
		"GitHub Git.Git-Extra     1.0     winget\n"
	_, a := matchRunner(t, table, nil)
	obs := a.LookupInstalled(context.Background(), "Git.Git")
	if obs.State != plan.StateUnknown {
		t.Errorf("substring row must be unknown, got %s", obs.State)
	}
}

func TestLookupNameWithSpaces(t *testing.T) {
	table := "Name               Id                         Version Available Source\n" +
		"------------------ -------------------------- ------- --------- ------\n" +
		"Visual Studio Code Microsoft.VisualStudioCode 1.90.0  1.91.0    winget\n"
	_, a := matchRunner(t, table, nil)
	obs := a.LookupInstalled(context.Background(), "Microsoft.VisualStudioCode")
	if obs.State != plan.StateSatisfied || !strings.Contains(obs.Detail, "1.90.0") {
		t.Errorf("expected satisfied with version, got %s (%s)", obs.State, obs.Detail)
	}
	if obs.Version != "1.90.0" {
		t.Errorf("expected version 1.90.0, got %q", obs.Version)
	}
}

func TestLookupFalseMatchAvoidance(t *testing.T) {
	table := "Name     Id        Version Source\n" +
		"---------------------------------\n" +
		"Git.Git  Other.App 1.0.0   winget\n"
	_, a := matchRunner(t, table, nil)

	// Searching for Git.Git (which is in the Name column, not Id) must NOT match.
	obsGit := a.LookupInstalled(context.Background(), "Git.Git")
	if obsGit.State != plan.StateUnknown {
		t.Errorf("lookup for ID in Name column must be unknown, got %s", obsGit.State)
	}

	// Searching for Other.App (in the Id column) must match.
	obsOther := a.LookupInstalled(context.Background(), "Other.App")
	if obsOther.State != plan.StateSatisfied {
		t.Fatalf("expected satisfied for Other.App, got %s (%s)", obsOther.State, obsOther.Detail)
	}
	if obsOther.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %q", obsOther.Version)
	}
}

func TestLookupAmbiguousHeader(t *testing.T) {
	// Missing separator line
	noSep := "Name Id Version\nGit Git.Git 2.54.0\n"
	_, a := matchRunner(t, noSep, nil)
	if obs := a.LookupInstalled(context.Background(), "Git.Git"); obs.State != plan.StateUnknown {
		t.Errorf("missing separator must be unknown, got %s", obs.State)
	}

	// Unaligned / unrecognizable header
	weirdHeader := "Output from winget:\n--------------\nGit.Git 2.54.0\n"
	_, b := matchRunner(t, weirdHeader, nil)
	if obs := b.LookupInstalled(context.Background(), "Git.Git"); obs.State != plan.StateUnknown {
		t.Errorf("unrecognizable header must be unknown, got %s", obs.State)
	}
}
func TestLookupLocalizedNoMatchIsUnknown(t *testing.T) {
	// Same exit code but an unrecognized (localized) message: absence is
	// not confirmed.
	_, a := matchRunner(t, "Keine installierte Pakete gefunden.\n",
		fakeExitError{code: 20, msg: "exit status 20"})
	obs := a.LookupInstalled(context.Background(), "No.Such.App")
	if obs.State != plan.StateUnknown {
		t.Errorf("localized no-match must be unknown, got %s", obs.State)
	}
}

func TestLookupUnexpectedExitCode(t *testing.T) {
	_, a := matchRunner(t, "weird output\n", fakeExitError{code: 1, msg: "exit status 1"})
	if obs := a.LookupInstalled(context.Background(), "Git.Git"); obs.State != plan.StateUnknown {
		t.Errorf("unexpected exit code must be unknown, got %s", obs.State)
	}
}

func TestLookupTimeoutAndFailure(t *testing.T) {
	_, a := matchRunner(t, "", context.DeadlineExceeded)
	if obs := a.LookupInstalled(context.Background(), "Git.Git"); obs.State != plan.StateUnknown {
		t.Errorf("timeout must be unknown, got %s", obs.State)
	}
	_, b := matchRunner(t, "partial", errors.New("boom"))
	if obs := b.LookupInstalled(context.Background(), "Git.Git"); obs.State != plan.StateUnknown {
		t.Errorf("command failure must be unknown, got %s", obs.State)
	}
}

func TestLookupEmptyID(t *testing.T) {
	runner, a := matchRunner(t, matchTable, nil)
	if obs := a.LookupInstalled(context.Background(), ""); obs.State != plan.StateUnknown {
		t.Errorf("empty id must be unknown, got %s", obs.State)
	}
	if len(runner.calls) != 0 {
		t.Error("empty id must not invoke winget at all")
	}
}

func TestInventoryCompatibility(t *testing.T) {
	_, a := matchRunner(t, matchTable, nil)
	var inv plan.Inventory = a
	obs := inv.LookupApplication("Git.Git", "winget")
	if obs.State != plan.StateSatisfied {
		t.Errorf("inventory lookup via adapter = %s, want satisfied", obs.State)
	}
	if obs.Version != "2.54.0" {
		t.Errorf("inventory lookup via adapter version = %s, want 2.54.0", obs.Version)
	}
	obsNonWinget := inv.LookupApplication("Git.Git", "msstore")
	if obsNonWinget.State != plan.StateUnknown || !strings.Contains(obsNonWinget.Detail, "msstore") {
		t.Errorf("non-winget source must be unknown, got %s (%s)", obsNonWinget.State, obsNonWinget.Detail)
	}
}

func TestNoFileModifications(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	runner, a := matchRunner(t, matchTable, nil)
	_ = a.ProbeAvailability(context.Background())
	_ = a.LookupInstalled(context.Background(), "Git.Git")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 || len(runner.calls) != 2 {
		t.Errorf("adapter must be side-effect free: dir=%v calls=%d", entries, len(runner.calls))
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
