package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testProfileDoc = `schema_version: 1
name: cli-test
platform:
  os: windows
  architectures: [amd64]
applications:
  - id: Git.Git
    source: winget
`

func writeTempProfile(t *testing.T, content string) string {
	t.Helper()
	return writeNamedTempProfile(t, "profile.yaml", content)
}

func writeNamedTempProfile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// withUserDir runs f with the CLI profile store pointed at dir.
func withUserDir(t *testing.T, dir string) {
	t.Helper()
	old := userDirOverride
	userDirOverride = dir
	t.Cleanup(func() { userDirOverride = old })
}

func runArgs(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

func TestProfileValidate_OK(t *testing.T) {
	path := writeTempProfile(t, testProfileDoc)
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"profile", "validate", "--file", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute(profile validate) returned error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"cli-test" is valid`) {
		t.Errorf("validate output missing success marker, got:\n%s", out)
	}
}

func TestProfileValidate_InvalidFile(t *testing.T) {
	path := writeTempProfile(t, "schema_version: 2\n")
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"profile", "validate", "--file", path})

	if err := cmd.Execute(); err == nil {
		t.Fatal("Execute(profile validate) with invalid file expected error, got nil")
	}
}

func TestProfileValidate_MissingFileFlag(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"profile", "validate"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("Execute(profile validate) without --file expected error, got nil")
	}
}

func TestProfileValidate_MissingPath(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"profile", "validate", "--file", filepath.Join(t.TempDir(), "nope.yaml")})

	if err := cmd.Execute(); err == nil {
		t.Fatal("Execute(profile validate) with missing path expected error, got nil")
	}
}

func TestProfileList_BuiltinAndLocal(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mine.yaml"), []byte(testProfileDoc), 0o600); err != nil {
		t.Fatal(err)
	}
	withUserDir(t, dir)

	out, err := runArgs(t, "profile", "list")
	if err != nil {
		t.Fatalf("Execute(profile list) returned error: %v", err)
	}
	for _, want := range []string{"estzdev", "builtin", "cli-test", "local", "ok"} {
		if !strings.Contains(out, want) {
			t.Errorf("list output missing %q, got:\n%s", want, out)
		}
	}
}

func TestProfileList_InvalidAndCollision(t *testing.T) {
	dir := t.TempDir()
	dup := func(name string) string {
		return "schema_version: 1\nname: " + name + "\nplatform:\n  os: windows\n  architectures: [amd64]\n"
	}
	if err := os.WriteFile(filepath.Join(dir, "a.yaml"), []byte(dup("Twin")), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.yaml"), []byte(dup("TWIN")), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte("schema_version: 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	withUserDir(t, dir)

	out, err := runArgs(t, "profile", "list")
	if err != nil {
		t.Fatalf("Execute(profile list) returned error: %v", err)
	}
	for _, want := range []string{"collision", "invalid"} {
		if !strings.Contains(out, want) {
			t.Errorf("list output missing %q, got:\n%s", want, out)
		}
	}
}

func TestProfileShow_Builtin(t *testing.T) {
	withUserDir(t, t.TempDir())

	out, err := runArgs(t, "profile", "show", "builtin:estzdev")
	if err != nil {
		t.Fatalf("Execute(profile show) returned error: %v", err)
	}
	for _, want := range []string{"Name: estzdev", "Source: builtin", "Git.Git", "Applications (2):"} {
		if !strings.Contains(out, want) {
			t.Errorf("show output missing %q, got:\n%s", want, out)
		}
	}
}

func TestProfileShow_LocalPreferred(t *testing.T) {
	dir := t.TempDir()
	local := "schema_version: 1\nname: ESTZDEV\ndescription: local copy\nplatform:\n  os: windows\n  architectures: [amd64]\n"
	if err := os.WriteFile(filepath.Join(dir, "mine.yaml"), []byte(local), 0o600); err != nil {
		t.Fatal(err)
	}
	withUserDir(t, dir)

	out, err := runArgs(t, "profile", "show", "estzdev")
	if err != nil {
		t.Fatalf("Execute(profile show) returned error: %v", err)
	}
	if !strings.Contains(out, "Source: local") || !strings.Contains(out, "local copy") {
		t.Errorf("show should prefer the user-owned profile, got:\n%s", out)
	}
}

func TestProfileShow_Missing(t *testing.T) {
	withUserDir(t, t.TempDir())

	if _, err := runArgs(t, "profile", "show", "nope"); err == nil {
		t.Fatal("Execute(profile show) with missing name expected error, got nil")
	}
}

func TestProfileShow_Collision(t *testing.T) {
	dir := t.TempDir()
	dup := func(name string) string {
		return "schema_version: 1\nname: " + name + "\nplatform:\n  os: windows\n  architectures: [amd64]\n"
	}
	if err := os.WriteFile(filepath.Join(dir, "a.yaml"), []byte(dup("Twin")), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.yaml"), []byte(dup("TWIN")), 0o600); err != nil {
		t.Fatal(err)
	}
	withUserDir(t, dir)

	_, err := runArgs(t, "profile", "show", "twin")
	if err == nil || !strings.Contains(err.Error(), "multiple files") {
		t.Errorf("Execute(profile show) with collision expected ambiguity error, got %v", err)
	}
}
