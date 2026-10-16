package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validDoc = `schema_version: 1
name: estzdev
description: Starter Windows developer setup
platform:
  os: windows
  architectures: [amd64]
applications:
  - id: Git.Git
    source: winget
  - id: Microsoft.VisualStudioCode
    source: winget
    version: 1.90.0
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

func TestParse_Valid(t *testing.T) {
	p, err := Parse([]byte(validDoc))
	if err != nil {
		t.Fatalf("Parse(valid) returned error: %v", err)
	}
	if p.Name != "estzdev" || p.SchemaVersion != 1 {
		t.Errorf("unexpected profile identity: %+v", p)
	}
	s := p.Summarize()
	if s.Applications != 2 || s.Runtimes != 1 || s.Extensions != 1 || s.Settings != 1 {
		t.Errorf("unexpected summary: %+v", s)
	}
}

func TestParse_ShippedExample(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "estzdev.profile.yaml"))
	if err != nil {
		t.Fatalf("cannot read shipped example: %v", err)
	}
	if _, err := Parse(data); err != nil {
		t.Errorf("shipped example failed to parse: %v", err)
	}
}

func TestParse_Invalid(t *testing.T) {
	longName := strings.Repeat("a", 41)
	longDesc := strings.Repeat("d", 241)
	manyApps := "schema_version: 1\nname: big\nplatform:\n  os: windows\n  architectures: [amd64]\napplications:\n" +
		strings.Repeat("  - id: Foo.Bar\n    source: winget\n", MaxApplications+1)
	manyRuntimes := "schema_version: 1\nname: big\nplatform:\n  os: windows\n  architectures: [amd64]\nruntimes:\n" +
		strings.Repeat("  - name: node\n    manager: mise\n    version: 24.11.1\n", MaxRuntimes+1)

	cases := []struct {
		name string
		doc  string
		want string
	}{
		{"empty", ``, "empty"},
		{"whitespace", "  \n\t\n", "empty"},
		{"malformed", "schema_version: [1\n", "not valid YAML"},
		{"missingSchema", "name: qq\nplatform:\n  os: windows\n  architectures: [amd64]\n", "schema_version"},
		{"schemaTwo", "schema_version: 2\nname: qq\nplatform:\n  os: windows\n  architectures: [amd64]\n", "schema_version"},
		{"schemaString", "schema_version: one\nname: qq\nplatform:\n  os: windows\n  architectures: [amd64]\n", "schema v1"},
		{"unknownRoot", "schema_version: 1\nname: qq\nrun: evil\nplatform:\n  os: windows\n  architectures: [amd64]\n", "run"},
		{"unknownNested", "schema_version: 1\nname: qq\nplatform:\n  os: windows\n  architectures: [amd64]\n  extra: 1\n", "extra"},
		{"unknownAppField", "schema_version: 1\nname: qq\nplatform:\n  os: windows\n  architectures: [amd64]\napplications:\n  - id: Git.Git\n    source: winget\n    cmd: evil\n", "cmd"},
		{"multiDoc", "schema_version: 1\nname: qq\nplatform:\n  os: windows\n  architectures: [amd64]\n---\nfoo: bar\n", "one YAML document"},
		{"alias", "schema_version: 1\nname: &n x\nnick: *n\nplatform:\n  os: windows\n  architectures: [amd64]\n", "aliases"},
		{"dupKey", "schema_version: 1\nname: a\nname: b\nplatform:\n  os: windows\n  architectures: [amd64]\n", "duplicate key"},
		{"badNameDash", "schema_version: 1\nname: -x\nplatform:\n  os: windows\n  architectures: [amd64]\n", "name"},
		{"badNameShort", "schema_version: 1\nname: a\nplatform:\n  os: windows\n  architectures: [amd64]\n", "name"},
		{"badNameLong", "schema_version: 1\nname: " + longName + "\nplatform:\n  os: windows\n  architectures: [amd64]\n", "name"},
		{"badNameSpace", "schema_version: 1\nname: \"has space\"\nplatform:\n  os: windows\n  architectures: [amd64]\n", "name"},
		{"longDesc", "schema_version: 1\nname: qq\ndescription: \"" + longDesc + "\"\nplatform:\n  os: windows\n  architectures: [amd64]\n", "description"},
		{"wrongOS", "schema_version: 1\nname: qq\nplatform:\n  os: linux\n  architectures: [amd64]\n", "platform.os"},
		{"noArch", "schema_version: 1\nname: qq\nplatform:\n  os: windows\n  architectures: []\n", "architectures"},
		{"badArch", "schema_version: 1\nname: qq\nplatform:\n  os: windows\n  architectures: [arm64]\n", "arm64"},
		{"dupArch", "schema_version: 1\nname: qq\nplatform:\n  os: windows\n  architectures: [amd64, amd64]\n", "duplicate"},
		{"badSource", "schema_version: 1\nname: qq\nplatform:\n  os: windows\n  architectures: [amd64]\napplications:\n  - id: Git.Git\n    source: https://evil.example/x.exe\n", "source"},
		{"shellID", "schema_version: 1\nname: qq\nplatform:\n  os: windows\n  architectures: [amd64]\napplications:\n  - id: \"Git.Git; rm -rf\"\n    source: winget\n", "id"},
		{"pipeID", "schema_version: 1\nname: qq\nplatform:\n  os: windows\n  architectures: [amd64]\napplications:\n  - id: \"a|b\"\n    source: winget\n", "id"},
		{"dupApps", "schema_version: 1\nname: qq\nplatform:\n  os: windows\n  architectures: [amd64]\napplications:\n  - id: Git.Git\n    source: winget\n  - id: git.git\n    source: winget\n", "duplicate"},
		{"tooManyApps", manyApps, "exceeds 100"},
		{"runtimeNoVersion", "schema_version: 1\nname: qq\nplatform:\n  os: windows\n  architectures: [amd64]\nruntimes:\n  - name: node\n    manager: mise\n", "version is required"},
		{"runtimeLatest", "schema_version: 1\nname: qq\nplatform:\n  os: windows\n  architectures: [amd64]\nruntimes:\n  - name: node\n    manager: mise\n    version: latest\n", "not an exact version"},
		{"runtimeStar", "schema_version: 1\nname: qq\nplatform:\n  os: windows\n  architectures: [amd64]\nruntimes:\n  - name: node\n    manager: mise\n    version: \"*\"\n", "not an exact version"},
		{"runtimeManager", "schema_version: 1\nname: qq\nplatform:\n  os: windows\n  architectures: [amd64]\nruntimes:\n  - name: node\n    manager: brew\n    version: 24.11.1\n", "manager"},
		{"runtimeCatalog", "schema_version: 1\nname: qq\nplatform:\n  os: windows\n  architectures: [amd64]\nruntimes:\n  - name: https://evil.example/plugin\n    manager: mise\n    version: 1.0.0\n", "catalog"},
		{"dupRuntimes", "schema_version: 1\nname: qq\nplatform:\n  os: windows\n  architectures: [amd64]\nruntimes:\n  - name: node\n    manager: mise\n    version: 24.11.1\n  - name: Node\n    manager: mise\n    version: 22.0.0\n", "duplicate"},
		{"tooManyRuntimes", manyRuntimes, "exceeds 30"},
		{"badEditor", "schema_version: 1\nname: qq\nplatform:\n  os: windows\n  architectures: [amd64]\neditor_extensions:\n  - editor: vim\n    id: foo.bar\n", "editor"},
		{"badExtID", "schema_version: 1\nname: qq\nplatform:\n  os: windows\n  architectures: [amd64]\neditor_extensions:\n  - editor: vscode\n    id: \"evil; id\"\n", "id"},
		{"badSettingKind", "schema_version: 1\nname: qq\nplatform:\n  os: windows\n  architectures: [amd64]\nsettings:\n  - kind: raw.exec\n    key: a\n    value: b\n", "kind"},
		{"badSettingKey", "schema_version: 1\nname: qq\nplatform:\n  os: windows\n  architectures: [amd64]\nsettings:\n  - kind: git.config\n    key: \"a b\"\n    value: c\n", "key"},
		{"oversized", strings.Repeat("#", MaxProfileBytes+1), "exceeds"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.doc))
			if err == nil {
				t.Fatalf("Parse(%s) expected error, got nil", tc.name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Errorf("Parse(%s) error %q does not contain %q", tc.name, err, tc.want)
			}
		})
	}
}

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()

	valid := filepath.Join(dir, "good.yaml")
	if err := os.WriteFile(valid, []byte(validDoc), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFile(valid); err != nil {
		t.Errorf("LoadFile(valid) returned error: %v", err)
	}

	if _, err := LoadFile(filepath.Join(dir, "missing.yaml")); err == nil {
		t.Error("LoadFile(missing) expected error, got nil")
	}
	if _, err := LoadFile(dir); err == nil {
		t.Error("LoadFile(directory) expected error, got nil")
	}
	if _, err := LoadFile(""); err == nil {
		t.Error("LoadFile(empty path) expected error, got nil")
	}

	bad := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(bad, []byte("schema_version: 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFile(bad); err == nil {
		t.Error("LoadFile(invalid content) expected error, got nil")
	}

	big := filepath.Join(dir, "big.yaml")
	f, err := os.Create(big)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(make([]byte, MaxProfileBytes+1)); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, err := LoadFile(big); err == nil {
		t.Error("LoadFile(oversized) expected error, got nil")
	}
}
