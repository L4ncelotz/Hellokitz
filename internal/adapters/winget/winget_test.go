package winget

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"hellokit/internal/plan"
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
	state, detail := a.LookupInstalled(context.Background(), "Git.Git")
	if state != plan.StateSatisfied {
		t.Fatalf("expected satisfied, got %s (%s)", state, detail)
	}
	if !strings.Contains(detail, "2.54.0") {
		t.Errorf("detail should carry detected version, got %q", detail)
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
	state, detail := a.LookupInstalled(context.Background(), "No.Such.App")
	if state != plan.StateMissing {
		t.Errorf("expected missing, got %s (%s)", state, detail)
	}
}

func TestLookupCaseSensitive(t *testing.T) {
	// --exact is case-sensitive: a lowercase query against a cased row is
	// not a confirmed match and must not be reported missing either.
	_, a := matchRunner(t, matchTable, nil)
	state, _ := a.LookupInstalled(context.Background(), "git.git")
	if state != plan.StateUnknown {
		t.Errorf("case-variant id must be unknown, got %s", state)
	}
}

func TestLookupSubstringIsNotMatch(t *testing.T) {
	table := "Name Id Version Source\n--------------------\nGitHub Git.Git-Extra 1.0 winget\n"
	_, a := matchRunner(t, table, nil)
	state, _ := a.LookupInstalled(context.Background(), "Git.Git")
	if state != plan.StateUnknown {
		t.Errorf("substring row must be unknown, got %s", state)
	}
}

func TestLookupNameWithSpaces(t *testing.T) {
	table := "Name Id Version Source\n--------------------\nVisual Studio Code Microsoft.VisualStudioCode 1.90.0 winget\n"
	_, a := matchRunner(t, table, nil)
	state, detail := a.LookupInstalled(context.Background(), "Microsoft.VisualStudioCode")
	if state != plan.StateSatisfied || !strings.Contains(detail, "1.90.0") {
		t.Errorf("expected satisfied with version, got %s (%s)", state, detail)
	}
}

func TestLookupLocalizedNoMatchIsUnknown(t *testing.T) {
	// Same exit code but an unrecognized (localized) message: absence is
	// not confirmed.
	_, a := matchRunner(t, "Keine installierte Pakete gefunden.\n",
		fakeExitError{code: 20, msg: "exit status 20"})
	state, _ := a.LookupInstalled(context.Background(), "No.Such.App")
	if state != plan.StateUnknown {
		t.Errorf("localized no-match must be unknown, got %s", state)
	}
}

func TestLookupUnexpectedExitCode(t *testing.T) {
	_, a := matchRunner(t, "weird output\n", fakeExitError{code: 1, msg: "exit status 1"})
	if state, _ := a.LookupInstalled(context.Background(), "Git.Git"); state != plan.StateUnknown {
		t.Errorf("unexpected exit code must be unknown, got %s", state)
	}
}

func TestLookupTimeoutAndFailure(t *testing.T) {
	_, a := matchRunner(t, "", context.DeadlineExceeded)
	if state, _ := a.LookupInstalled(context.Background(), "Git.Git"); state != plan.StateUnknown {
		t.Errorf("timeout must be unknown, got %s", state)
	}
	_, b := matchRunner(t, "partial", errors.New("boom"))
	if state, _ := b.LookupInstalled(context.Background(), "Git.Git"); state != plan.StateUnknown {
		t.Errorf("command failure must be unknown, got %s", state)
	}
}

func TestLookupEmptyID(t *testing.T) {
	runner, a := matchRunner(t, matchTable, nil)
	if state, _ := a.LookupInstalled(context.Background(), ""); state != plan.StateUnknown {
		t.Errorf("empty id must be unknown, got %s", state)
	}
	if len(runner.calls) != 0 {
		t.Error("empty id must not invoke winget at all")
	}
}

func TestInventoryCompatibility(t *testing.T) {
	_, a := matchRunner(t, matchTable, nil)
	var inv plan.Inventory = a
	state, _ := inv.LookupApplication("Git.Git", "winget")
	if state != plan.StateSatisfied {
		t.Errorf("inventory lookup via adapter = %s, want satisfied", state)
	}
	state, detail := inv.LookupApplication("Git.Git", "msstore")
	if state != plan.StateUnknown || !strings.Contains(detail, "msstore") {
		t.Errorf("non-winget source must be unknown, got %s (%s)", state, detail)
	}
}

func TestNoFileModifications(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	runner, a := matchRunner(t, matchTable, nil)
	_ = a.ProbeAvailability(context.Background())
	_, _ = a.LookupInstalled(context.Background(), "Git.Git")
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
