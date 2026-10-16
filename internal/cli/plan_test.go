package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanCmd_ReadOnlyPreview(t *testing.T) {
	path := writeTempProfile(t, testProfileDoc)
	withUserDir(t, t.TempDir())

	out, err := runArgs(t, "plan", "--file", path)
	if err != nil {
		t.Fatalf("Execute(plan) returned error: %v", err)
	}
	for _, want := range []string{
		`Plan for profile "cli-test" (windows/amd64):`,
		"[UNKNOWN] Git.Git",
		"Cannot verify",
		"No changes have been made.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("plan output missing %q, got:\n%s", want, out)
		}
	}
}

func TestPlanCmd_InvalidFile(t *testing.T) {
	path := writeTempProfile(t, "schema_version: 2\n")

	if _, err := runArgs(t, "plan", "--file", path); err == nil {
		t.Fatal("Execute(plan) with invalid file expected error, got nil")
	}
}

func TestPlanCmd_MissingFileFlag(t *testing.T) {
	if _, err := runArgs(t, "plan"); err == nil {
		t.Fatal("Execute(plan) without --file expected error, got nil")
	}
}

func TestPlanCmd_MissingPath(t *testing.T) {
	if _, err := runArgs(t, "plan", "--file", filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("Execute(plan) with missing path expected error, got nil")
	}
}

func TestPlanCmd_DoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.yaml")
	if err := os.WriteFile(path, []byte(testProfileDoc), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runArgs(t, "plan", "--file", path); err != nil {
		t.Fatalf("Execute(plan) returned error: %v", err)
	}
	after, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != len(after) {
		t.Errorf("plan created files in the profile directory: %v -> %v", before, after)
	}
}
