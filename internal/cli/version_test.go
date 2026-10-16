package cli

import (
	"bytes"
	"encoding/json"
	"runtime"
	"strings"
	"testing"

	"hellokit/internal/version"
)

func executeRoot(args ...string) (string, error) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

func TestVersionCmd_Text(t *testing.T) {
	out, err := executeRoot("version")
	if err != nil {
		t.Fatalf("Execute(version) returned error: %v", err)
	}
	if !strings.Contains(out, version.Version) {
		t.Errorf("version output missing %q, got:\n%s", version.Version, out)
	}
	if !strings.Contains(out, runtime.GOOS) {
		t.Errorf("version output missing OS %q, got:\n%s", runtime.GOOS, out)
	}
}

func TestVersionCmd_JSON(t *testing.T) {
	out, err := executeRoot("version", "--json")
	if err != nil {
		t.Fatalf("Execute(version --json) returned error: %v", err)
	}

	var info version.Info
	if err := json.Unmarshal([]byte(out), &info); err != nil {
		t.Fatalf("version --json output is not valid JSON: %v\noutput:\n%s", err, out)
	}
	if info.Version != version.Version {
		t.Errorf("JSON version = %q, want %q", info.Version, version.Version)
	}
	if !strings.Contains(out, `"version": "0.1.0-dev"`) {
		t.Errorf("JSON output missing version field, got:\n%s", out)
	}
}
