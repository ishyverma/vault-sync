package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommand_Version(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "VaultSync") {
		t.Errorf("expected version output to contain 'VaultSync', got: %s", output)
	}
	if !strings.Contains(output, "0.1.0-dev") {
		t.Errorf("expected version output to contain '0.1.0-dev', got: %s", output)
	}
}

func TestRootCommand_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	expectedSections := []string{
		"VaultSync",
		"vim my-note.md",
		"Usage:",
		"Flags:",
		"--help",
		"--version",
	}
	for _, section := range expectedSections {
		if !strings.Contains(output, section) {
			t.Errorf("expected help output to contain %q, got: %s", section, output)
		}
	}
}

func TestRootCommand_NoArgsShowsHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Usage:") {
		t.Errorf("expected no-args to show help, got: %s", output)
	}
}
