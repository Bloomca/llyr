package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetToolOptionsUsesSafeNonInteractiveDefaults(t *testing.T) {
	tests := []struct {
		tool string
		want string
	}{
		{tool: "claude", want: "claude -p --permission-mode auto"},
		{tool: "codex", want: "codex -a never -s read-only exec"},
		{tool: "pi", want: "pi --no-approve --tools read,bash,grep,find,ls -p"},
		{tool: "custom-agent", want: "custom-agent"},
	}

	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			if got := getToolOptions(tt.tool); got != tt.want {
				t.Fatalf("getToolOptions(%q) = %q, want %q", tt.tool, got, tt.want)
			}
		})
	}
}

func TestReadAgentToolValidatesAndReturnsCommand(t *testing.T) {
	dir := t.TempDir()
	toolPath := filepath.Join(dir, "custom-agent")
	if err := os.WriteFile(toolPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	var output bytes.Buffer
	const command = `custom-agent --mode "code review"`
	got, err := readAgentTool(strings.NewReader(command+"\n"), &output)
	if err != nil {
		t.Fatalf("readAgentTool() error = %v", err)
	}
	if got != command {
		t.Fatalf("readAgentTool() = %q, want %q", got, command)
	}
	if output.String() != "Enter the CLI command and any arguments: " {
		t.Fatalf("prompt = %q", output.String())
	}
}

func TestReadAgentToolRejectsInvalidCommands(t *testing.T) {
	dir := t.TempDir()
	toolPath := filepath.Join(dir, "custom-agent")
	if err := os.WriteFile(toolPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	tests := []struct {
		name    string
		line    string
		wantErr string
	}{
		{name: "empty", line: "\n", wantErr: "empty command"},
		{name: "unterminated quote", line: "custom-agent \"review\n", wantErr: "unterminated quote"},
		{name: "trailing backslash", line: "custom-agent \\\n", wantErr: "trailing backslash"},
		{name: "missing executable", line: "missing-agent --review\n", wantErr: `look up CLI tool "missing-agent"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := readAgentTool(strings.NewReader(tt.line), &bytes.Buffer{})
			if err == nil {
				t.Fatal("readAgentTool() error = nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("readAgentTool() error = %q, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}
