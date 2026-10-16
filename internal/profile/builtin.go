package profile

import (
	_ "embed"
)

// builtinEstzdev is the bundled read-only sample profile. The canonical
// editable sample lives at examples/estzdev.profile.yaml; builtin_test.go
// asserts the two stay byte-identical so they cannot drift apart silently.
//
//go:embed builtin/estzdev.profile.yaml
var builtinEstzdev []byte

// DefaultBuiltins returns the bundled read-only sample profiles.
func DefaultBuiltins() []Builtin {
	return []Builtin{{Data: builtinEstzdev}}
}
