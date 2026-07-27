package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func review(c config, dir string) {
	cmd := executeCommand(dir, c.AgentTool, constructPrompt())
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			fmt.Printf("%s exited %d: %s", c.AgentTool, ee.ExitCode(), stderr.String())
			os.Exit(1)
		}
		fmt.Printf("%s failed with error: %s", c.AgentTool, err)
		os.Exit(1)
	}

	// parse response and post to GH as a review
	fmt.Print(strings.TrimSpace(stdout.String()))
}

func constructPrompt() string {
	prompt := `
Check this PR and compare the difference with the branch it points against.
Read the README.md file (if any), documentation and check the source code.

Once you have a good understanding of the project, go ahead and review the changes.
Provide the output in a JSON format with the following schema:
{
  overview: string
  feedback: { level: 'p1' | 'p2' | 'p3'; file: string; line: number }[]
}
`

	return strings.TrimSpace(prompt)
}
