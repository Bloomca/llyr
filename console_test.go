package main

import (
	"bytes"
	"fmt"
	"testing"
)

func TestColorWriterWrapsOutput(t *testing.T) {
	var output bytes.Buffer
	writer := colorWriter{
		out:     &output,
		color:   toolOutputColor,
		enabled: true,
	}

	input := []byte("tool output\n")
	written, err := writer.Write(input)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if written != len(input) {
		t.Fatalf("Write() wrote %d bytes, want %d", written, len(input))
	}

	want := toolOutputColor + string(input) + ansiReset
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

func TestPrintActionToStylesAndSpacesOutput(t *testing.T) {
	var output bytes.Buffer
	printActionTo(&output, true, "Reviewing %s", "owner/repo#1")

	want := ansiItalic + actionColor + "Reviewing owner/repo#1\n" + ansiReset + "\n"
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

func TestPrintActionToLeavesPlainOutputSpaced(t *testing.T) {
	var output bytes.Buffer
	printActionTo(&output, false, "Preparing diff")

	const want = "Preparing diff\n\n"
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

func TestToolOutputWriterLeavesNonTerminalOutputPlain(t *testing.T) {
	var output bytes.Buffer
	if _, err := fmt.Fprint(toolOutputWriter(&output), "tool output\n"); err != nil {
		t.Fatalf("writing tool output: %v", err)
	}

	const want = "tool output\n"
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}
