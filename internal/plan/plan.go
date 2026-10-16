// Package plan builds immutable, read-only execution previews from validated
// profiles and an inventory.
//
// A Plan never mutates the machine and never claims more than the inventory
// can verify: anything unverifiable is reported as unknown, never assumed
// installed. Real detection adapters land in later milestones; until then
// the CLI builds plans with UnknownInventory.
package plan

import (
	"fmt"
	"strings"

	"hellokit/internal/profile"
)

// Action states carried by a plan.
const (
	StateSatisfied = "satisfied"
	StateMissing   = "missing"
	StateMismatch  = "mismatch"
	StateUnknown   = "unknown"
	StateBlocked   = "blocked"
)

// Action kinds carried by a plan.
const (
	KindApplication = "application"
	KindRuntime     = "runtime"
	KindExtension   = "extension"
	KindSetting     = "setting"
)

// Action is one desired-state item with its detected state.
type Action struct {
	Kind    string
	Target  string
	Source  string
	Desired string
	State   string
	Reason  string
}

// Plan is an immutable preview of the changes a profile implies.
type Plan struct {
	ProfileName string
	Platform    string
	Actions     []Action
}

// Inventory reports detected installation state. Implementations must
// distinguish a successful query from an inability to query.
type Inventory interface {
	// LookupApplication returns the detected state and a short detail for
	// one application. Unknown states must explain what could not be verified.
	LookupApplication(id, source string) (state, detail string)
}

// UnknownInventory reports every application as unknown. The CLI uses it
// until real detection adapters exist; it is always safe because it claims
// nothing about the machine.
type UnknownInventory struct{}

// LookupApplication implements Inventory.
func (UnknownInventory) LookupApplication(_, _ string) (string, string) {
	return StateUnknown, "installation state cannot be determined yet (detection lands in M2)"
}

// Build derives a deterministic preview from a validated profile. Entry
// order follows the profile; no timestamps are recorded so identical
// profiles always produce identical plans.
func Build(p *profile.Profile, inv Inventory) Plan {
	pl := Plan{
		ProfileName: p.Name,
		Platform:    p.Platform.OS + "/" + strings.Join(p.Platform.Architectures, ","),
	}
	for _, a := range p.Applications {
		state, detail := inv.LookupApplication(a.ID, a.Source)
		pl.Actions = append(pl.Actions, Action{
			Kind:    KindApplication,
			Target:  a.ID,
			Source:  a.Source,
			Desired: a.Version,
			State:   state,
			Reason:  detail,
		})
	}
	for _, r := range p.Runtimes {
		pl.Actions = append(pl.Actions, Action{
			Kind:    KindRuntime,
			Target:  r.Name,
			Source:  r.Manager,
			Desired: r.Version,
			State:   StateUnknown,
			Reason:  fmt.Sprintf("runtime detection lands in M2/M4; %s %s is assumed neither installed nor missing", r.Name, r.Version),
		})
	}
	for _, e := range p.EditorExtensions {
		pl.Actions = append(pl.Actions, Action{
			Kind:   KindExtension,
			Target: e.ID,
			Source: e.Editor,
			State:  StateUnknown,
			Reason: "extension detection lands in M4; assumed neither installed nor missing",
		})
	}
	for _, s := range p.Settings {
		pl.Actions = append(pl.Actions, Action{
			Kind:    KindSetting,
			Target:  s.Kind + " " + s.Key,
			Source:  s.Kind,
			Desired: s.Value,
			State:   StateUnknown,
			Reason:  "configuration detection lands in M4; assumed neither applied nor missing",
		})
	}
	return pl
}
