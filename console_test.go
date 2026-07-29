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
