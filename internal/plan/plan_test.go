package plan

import (
	"strings"
	"testing"

	"hellokit/internal/profile"
)

// fakeInventory is the deterministic test inventory required by M1.
type fakeInventory struct {
	apps map[string]Observation // lower(source\x00id) -> Observation
}

func (f fakeInventory) LookupApplication(id, source string) Observation {
	if v, ok := f.apps[strings.ToLower(source+"\x00"+id)]; ok {
		return v
	}
	return Observation{State: StateUnknown, Detail: "not covered by fake inventory"}
}

func mustParse(t *testing.T, doc string) *profile.Profile {
	t.Helper()
	p, err := profile.Parse([]byte(doc))
	if err != nil {
		t.Fatalf("Parse test fixture failed: %v", err)
	}
	return p
}

const planDoc = `schema_version: 1
name: plan-test
platform:
  os: windows
  architectures: [amd64]
applications:
  - id: Git.Git
    source: winget
  - id: Missing.App
    source: winget
  - id: Stale.App
    source: winget
    version: 2.0.0
  - id: Blocked.App
    source: winget
runtimes:
  - name: node
    manager: mise
    version: 24.11.1
editor_extensions:
  - editor: vscode
    id: esbenp.prettier-vscode
settings:
  - kind: git.config
    key: init.defaultBranch
    value: main
`

func TestBuild_StateMapping(t *testing.T) {
	inv := fakeInventory{apps: map[string]Observation{
		"winget\x00git.git":     {State: StateSatisfied, Version: "1.0", Detail: "1.0 installed"},
		"winget\x00missing.app": {State: StateMissing, Detail: "not installed"},
		"winget\x00stale.app":   {State: StateSatisfied, Version: "1.0", Detail: "1.0 installed"},
		"winget\x00blocked.app": {State: StateBlocked, Detail: "needs elevation"},
	}}
	pl := Build(mustParse(t, planDoc), inv)
	if pl.ProfileName != "plan-test" || pl.Platform != "windows/amd64" {
		t.Fatalf("unexpected plan identity: %+v", pl)
	}
	if len(pl.Actions) != 7 {
		t.Fatalf("expected 7 actions, got %+v", pl.Actions)
	}
	wantStates := []string{
		StateSatisfied, StateMissing, StateMismatch, StateBlocked,
		StateUnknown, StateUnknown, StateUnknown,
	}
	for i, want := range wantStates {
		if pl.Actions[i].State != want {
			t.Errorf("action %d (%s) state = %s, want %s", i, pl.Actions[i].Target, pl.Actions[i].State, want)
		}
	}
	if pl.Actions[0].Reason != "1.0 installed" {
		t.Errorf("inventory detail not carried through: %+v", pl.Actions[0])
	}
	if pl.Actions[2].Desired != "2.0.0" {
		t.Errorf("desired version not carried through: %+v", pl.Actions[2])
	}
}

func TestBuild_UnknownInventoryClaimsNothing(t *testing.T) {
	pl := Build(mustParse(t, planDoc), UnknownInventory{})
	for _, a := range pl.Actions {
		if a.Kind == KindApplication && a.State != StateUnknown {
			t.Errorf("UnknownInventory must report unknown, got %+v", a)
		}
		if a.Reason == "" {
			t.Errorf("every action needs a reason, got %+v", a)
		}
	}
}

func TestBuild_EmptyProfile(t *testing.T) {
	doc := "schema_version: 1\nname: empty-one\nplatform:\n  os: windows\n  architectures: [amd64]\n"
	pl := Build(mustParse(t, doc), UnknownInventory{})
	if len(pl.Actions) != 0 {
		t.Errorf("expected no actions, got %+v", pl.Actions)
	}
}

func TestBuild_Deterministic(t *testing.T) {
	p := mustParse(t, planDoc)
	a := Build(p, UnknownInventory{})
	b := Build(mustParse(t, planDoc), UnknownInventory{})
	if len(a.Actions) != len(b.Actions) {
		t.Fatal("non-deterministic action count")
	}
	for i := range a.Actions {
		if a.Actions[i] != b.Actions[i] {
			t.Fatalf("non-deterministic action %d: %+v vs %+v", i, a.Actions[i], b.Actions[i])
		}
	}
}

func TestBuild_VersionComparison(t *testing.T) {
	doc := `schema_version: 1
name: ver-test
platform:
  os: windows
  architectures: [amd64]
applications:
  - id: Match.App
    source: winget
    version: 2.0.0
  - id: Mismatch.App
    source: winget
    version: 2.0.0
  - id: UnknownVer.App
    source: winget
    version: 2.0.0
  - id: Unpinned.App
    source: winget
  - id: MissingPinned.App
    source: winget
    version: 3.0.0
`
	inv := fakeInventory{apps: map[string]Observation{
		"winget\x00match.app":         {State: StateSatisfied, Version: "2.0.0", Detail: "v2.0.0"},
		"winget\x00mismatch.app":      {State: StateSatisfied, Version: "1.5.0", Detail: "v1.5.0"},
		"winget\x00unknownver.app":    {State: StateSatisfied, Version: "", Detail: "installed"},
		"winget\x00unpinned.app":      {State: StateSatisfied, Version: "4.0.0", Detail: "v4.0.0"},
		"winget\x00missingpinned.app": {State: StateMissing, Detail: "absent"},
	}}

	pl := Build(mustParse(t, doc), inv)
	if len(pl.Actions) != 5 {
		t.Fatalf("expected 5 actions, got %d", len(pl.Actions))
	}

	// Match.App: pinned matches installed
	if pl.Actions[0].State != StateSatisfied {
		t.Errorf("Match.App: got state %s, want %s", pl.Actions[0].State, StateSatisfied)
	}
	if !strings.Contains(pl.Actions[0].Reason, "matches desired version 2.0.0") {
		t.Errorf("Match.App: unexpected reason %q", pl.Actions[0].Reason)
	}

	// Mismatch.App: pinned differs from installed
	if pl.Actions[1].State != StateMismatch {
		t.Errorf("Mismatch.App: got state %s, want %s", pl.Actions[1].State, StateMismatch)
	}
	if !strings.Contains(pl.Actions[1].Reason, "does not match desired version 2.0.0") {
		t.Errorf("Mismatch.App: unexpected reason %q", pl.Actions[1].Reason)
	}

	// UnknownVer.App: pinned but installed version empty
	if pl.Actions[2].State != StateUnknown {
		t.Errorf("UnknownVer.App: got state %s, want %s", pl.Actions[2].State, StateUnknown)
	}
	if !strings.Contains(pl.Actions[2].Reason, "version cannot be verified against desired 2.0.0") {
		t.Errorf("UnknownVer.App: unexpected reason %q", pl.Actions[2].Reason)
	}

	// Unpinned.App: floating version, stays satisfied with adapter detail
	if pl.Actions[3].State != StateSatisfied {
		t.Errorf("Unpinned.App: got state %s, want %s", pl.Actions[3].State, StateSatisfied)
	}
	if pl.Actions[3].Reason != "v4.0.0" {
		t.Errorf("Unpinned.App: got reason %q, want v4.0.0", pl.Actions[3].Reason)
	}

	// MissingPinned.App: missing remains missing
	if pl.Actions[4].State != StateMissing {
		t.Errorf("MissingPinned.App: got state %s, want %s", pl.Actions[4].State, StateMissing)
	}
}
