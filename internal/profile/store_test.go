package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const storeValidDoc = `schema_version: 1
name: WORKSTATION
description: Test machine
platform:
  os: windows
  architectures: [amd64]
applications:
  - id: Git.Git
    source: winget
`

func writeProfile(t *testing.T, dir, file, content string) string {
	t.Helper()
	path := filepath.Join(dir, file)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestBuiltinParity(t *testing.T) {
	example, err := os.ReadFile(filepath.Join("..", "..", "examples", "estzdev.profile.yaml"))
	if err != nil {
		t.Fatalf("cannot read shipped example: %v", err)
	}
	if string(example) != string(builtinEstzdev) {
		t.Error("builtin sample drifted from examples/estzdev.profile.yaml; update both copies")
	}
	if _, err := Parse(builtinEstzdev); err != nil {
		t.Errorf("builtin sample is invalid: %v", err)
	}
}

func TestStore_ListEmptyDir(t *testing.T) {
	s := Store{UserDir: t.TempDir(), Builtins: DefaultBuiltins()}
	entries, err := s.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(entries) != 1 || entries[0].Source != SourceBuiltin || entries[0].Name != "estzdev" {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}

func TestStore_ListMissingDir(t *testing.T) {
	s := Store{UserDir: filepath.Join(t.TempDir(), "does-not-exist"), Builtins: DefaultBuiltins()}
	entries, err := s.List()
	if err != nil {
		t.Fatalf("List with missing dir returned error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected builtins only, got %+v", entries)
	}
}

func TestStore_ListInvalidAndIgnored(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "good.yaml", storeValidDoc)
	writeProfile(t, dir, "bad.yaml", "schema_version: 2\n")
	writeProfile(t, dir, "notes.txt", "not a profile\n")

	s := Store{UserDir: dir, Builtins: DefaultBuiltins()}
	entries, err := s.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	// builtin + good + bad; notes.txt ignored.
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %+v", entries)
	}
	var bad *Entry
	for i := range entries {
		if entries[i].File == "bad.yaml" {
			bad = &entries[i]
		}
	}
	if bad == nil || bad.Valid() {
		t.Fatalf("bad.yaml should be listed as invalid: %+v", entries)
	}
	if bad.Name != "" || bad.Profile != nil {
		t.Errorf("invalid entry must not expose name/profile: %+v", bad)
	}
}

func TestStore_UserCollisionCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	doc := func(name string) string {
		return "schema_version: 1\nname: " + name + "\nplatform:\n  os: windows\n  architectures: [amd64]\n"
	}
	writeProfile(t, dir, "a.yaml", doc("DevBox"))
	writeProfile(t, dir, "b.yaml", doc("devbox"))

	s := Store{UserDir: dir}
	entries, err := s.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %+v", entries)
	}
	for _, e := range entries {
		if !e.Collision {
			t.Errorf("expected collision flag on %+v", e)
		}
	}

	if _, _, err := s.Resolve("DEVBOX"); err == nil || !strings.Contains(err.Error(), "multiple files") {
		t.Errorf("Resolve(collision) expected ambiguity error, got %v", err)
	}
}

func TestStore_ResolvePrecedence(t *testing.T) {
	dir := t.TempDir()
	// Shadow the builtin estzdev with a user-owned profile of the same
	// (case-insensitively equal) name.
	shadow := "schema_version: 1\nname: ESTZDEV\ndescription: mine\nplatform:\n  os: windows\n  architectures: [amd64]\n"
	writeProfile(t, dir, "mine.yaml", shadow)

	s := Store{UserDir: dir, Builtins: DefaultBuiltins()}

	p, e, err := s.Resolve("estzdev")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if e.Source != SourceLocal || p.Description != "mine" {
		t.Errorf("user profile should win over builtin, got %+v", e)
	}

	p, e, err = s.Resolve("builtin:estzdev")
	if err != nil {
		t.Fatalf("Resolve(builtin:) returned error: %v", err)
	}
	if e.Source != SourceBuiltin {
		t.Errorf("builtin: prefix should select the sample, got %+v", e)
	}
	if p.Description != "" && strings.Contains(p.Description, "mine") {
		t.Errorf("builtin: resolution leaked user profile: %+v", p)
	}
}

func TestStore_ResolveMissing(t *testing.T) {
	s := Store{UserDir: t.TempDir(), Builtins: DefaultBuiltins()}
	if _, _, err := s.Resolve("nope"); err == nil {
		t.Error("Resolve(missing) expected error, got nil")
	}
	if _, _, err := s.Resolve("builtin:nope"); err == nil {
		t.Error("Resolve(builtin:missing) expected error, got nil")
	}
	if _, _, err := s.Resolve(""); err == nil {
		t.Error("Resolve(empty) expected error, got nil")
	}
}
