package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Feedback struct {
	Level string `json:"level"`
	File  string `json:"file"`
	Line  int    `json:"line"`
	Text  string `json:"text"`
}

type Review struct {
	Overview string     `json:"overview"`
	Feedback []Feedback `json:"feedback"`
}

type InlineComment struct {
	Path string `json:"path"`
	Body string `json:"body"`

	// Line is the line number in the file. Side is "RIGHT" for the new
	// version (the default) or "LEFT" for the original.
	Line int    `json:"line,omitempty"`
	Side string `json:"side,omitempty"`

	// StartLine and StartSide turn the comment into a multi-line range
	// ending at Line/Side.
	StartLine int    `json:"start_line,omitempty"`
	StartSide string `json:"start_side,omitempty"`
}

// JSON payload for GH API
type GitHubReview struct {
	Body     string          `json:"body,omitempty"`
	Event    string          `json:"event"`
	Comments []InlineComment `json:"comments,omitempty"`
}

func review(c config, dir string, pr pullRequest) {
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
	output := strings.TrimSpace(stdout.String())

	parsedReview := Review{}
	if err := json.Unmarshal([]byte(output), &parsedReview); err != nil {
		fmt.Printf("Failed to parse the output:\n%s", output)
		os.Exit(1)
	}

	postReview(dir, pr, parsedReview)
}

func constructPrompt() string {
	prompt := `
Check this PR and compare the difference with the branch it points against.
Read the README.md file (if any), documentation and check the source code.

Once you have a good understanding of the project, go ahead and review the changes.
Provide the output in a JSON format with the following schema:
{
  overview: string
  feedback: { level: 'p1' | 'p2' | 'p3'; file: string; line: number; text: string }[]
}
`

	return strings.TrimSpace(prompt)
}

func postReview(dir string, pr pullRequest, review Review) {
	ghReview := createGitHubReviewStruct(review)

	data, err := json.Marshal(ghReview)
	if err != nil {
		fmt.Printf("Could not encode GitHub review: %v\n", err)
		os.Exit(1)
	}

	endpoint := fmt.Sprintf("repos/%s/pulls/%d/reviews", pr.slug(), pr.number)
	ghStdin(
		dir,
		bytes.NewReader(data),
		"api",
		endpoint,
		"--method", "POST",
		"--input", "-",
	)
}

func ghStdin(dir string, stdin io.Reader, args ...string) {
	cmd := newGitHubCLI(dir, args)
	cmd.Stdin = stdin

	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			fmt.Printf("gh %s: %s", strings.Join(args, " "), msg)
			os.Exit(1)
		}

		fmt.Printf("gh %s: %v", strings.Join(args, " "), err)
		os.Exit(1)
	}
}

func createGitHubReviewStruct(review Review) GitHubReview {
	overview := strings.TrimSpace(review.Overview)
	if overview == "" {
		fmt.Println("Review overview cannot be empty")
		os.Exit(1)
	}

	comments := make([]InlineComment, 0, len(review.Feedback))

	for _, feedback := range review.Feedback {
		level := strings.ToLower(strings.TrimSpace(feedback.Level))
		switch level {
		case "p1", "p2", "p3":
		default:
			fmt.Printf("Invalid feedback level in the review %q\n", feedback.Level)
			os.Exit(1)
		}

		path := strings.TrimPrefix(strings.TrimSpace(feedback.File), "./")
		if path == "" || feedback.Line <= 0 || strings.TrimSpace(feedback.Text) == "" {
			fmt.Println("Invalid inline review feedback")
			os.Exit(1)
		}

		comments = append(comments, InlineComment{
			Path: path,
			Line: feedback.Line,
			Side: "RIGHT",
			Body: fmt.Sprintf(
				"**%s**: %s",
				strings.ToUpper(level),
				strings.TrimSpace(feedback.Text),
			),
		})
	}

	return GitHubReview{
		Body:     overview,
		Event:    "COMMENT",
		Comments: comments,
	}
}
