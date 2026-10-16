package plan

import (
	"strings"
	"testing"

	"hellokit/internal/profile"
)

// fakeInventory is the deterministic test inventory required by M1.
type fakeInventory struct {
	apps map[string][2]string // lower(source\x00id) -> (state, detail)
}

func (f fakeInventory) LookupApplication(id, source string) (string, string) {
	if v, ok := f.apps[strings.ToLower(source+"\x00"+id)]; ok {
		return v[0], v[1]
	}
	return StateUnknown, "not covered by fake inventory"
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
	inv := fakeInventory{apps: map[string][2]string{
		"winget\x00git.git":     {StateSatisfied, "1.0 installed"},
		"winget\x00missing.app": {StateMissing, "not installed"},
		"winget\x00stale.app":   {StateMismatch, "1.0 installed, want 2.0.0"},
		"winget\x00blocked.app": {StateBlocked, "needs elevation"},
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
