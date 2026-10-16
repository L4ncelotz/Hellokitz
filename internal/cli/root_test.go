package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCmd_Help(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute(--help) returned error: %v", err)
	}

	if out := buf.String(); !strings.Contains(out, "One command. Your dev environment.") {
		t.Errorf("help output missing tagline, got:\n%s", out)
	}
}

func TestRootCmd_UnknownFlag(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--bogus"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("Execute(--bogus) expected error, got nil")
	}
}
