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
	Side  string `json:"side"`
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
}

// JSON payload for GH API
type GitHubReview struct {
	Body     string          `json:"body,omitempty"`
	Event    string          `json:"event"`
	CommitID string          `json:"commit_id"`
	Comments []InlineComment `json:"comments,omitempty"`
}

const commentMarker = "> _Posted by [Llŷr](https://github.com/Bloomca/llyr)_"
const commentPrefix = commentMarker + "\n\n"

func review(c config, dir string, pr pullRequest) {
	printAction("Preparing pull request diff against %s", pr.baseRefName)
	commitID, err := captureHeadCommit(dir)
	if err != nil {
		fmt.Printf("Could not capture the checked-out commit: %v", err)
		os.Exit(1)
	}
	if commitID != pr.headCommitID {
		fmt.Printf(
			"Pull request changed during checkout: expected HEAD %s, got %s; please retry",
			pr.headCommitID,
			commitID,
		)
		os.Exit(1)
	}

	diff, err := capturePullRequestDiff(dir, pr.baseRefName, pr.baseCommitID, commitID)
	if err != nil {
		fmt.Printf("Could not capture the pull request diff: %v", err)
		os.Exit(1)
	}

	fileLabel := "files"
	if diff.changedFiles == 1 {
		fileLabel = "file"
	}
	printAction(
		"Diff size: %d LoC deleted, %d LoC added across %d %s",
		diff.deletedLines,
		diff.addedLines,
		diff.changedFiles,
		fileLabel,
	)

	cmd := executeCommand(
		dir,
		c.AgentTool,
		constructPrompt(pr.baseRefName, pr.baseCommitID, commitID),
	)
	printAction("Running review with %s", c.AgentTool)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			fmt.Printf("%s exited %d", c.AgentTool, ee.ExitCode())
			os.Exit(1)
		}
		fmt.Printf("%s failed with error: %s", c.AgentTool, err)
		os.Exit(1)
	}

	// parse response and post to GH as a review
	output := strings.TrimSpace(stdout.String())

	parsedReview, err := parseReviewOutput(output)
	if err != nil {
		fmt.Printf("Failed to parse the output as review JSON: %v\n%s\n", err, output)
		os.Exit(1)
	}

	printAction("Posting review to %s#%d", pr.slug(), pr.number)
	postReview(dir, pr, commitID, parsedReview, diff)
	printAction("Review posted successfully")
}

// Capture commit before doing a review so that the GH review is
// specifically pinned against that commit. This is needed in case
// there were new commits in between and that caused some changes
// to move in position.
func captureHeadCommit(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD^{commit}")
	cmd.Dir = dir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = toolOutputWriter(os.Stderr)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git rev-parse HEAD: %w", err)
	}

	commitID := strings.TrimSpace(stdout.String())
	if commitID == "" {
		return "", errors.New("git rev-parse HEAD returned an empty commit")
	}
	return commitID, nil
}

func parseReviewOutput(output string) (Review, error) {
	output = strings.TrimSpace(output)
	if unfenced, ok := unwrapJSONCodeFence(output); ok {
		output = unfenced
	}

	var review Review
	if err := json.Unmarshal([]byte(output), &review); err != nil {
		return Review{}, fmt.Errorf("decode review JSON: %w", err)
	}
	return review, nil
}

func unwrapJSONCodeFence(output string) (string, bool) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 3 || !strings.EqualFold(strings.TrimSpace(lines[0]), "```json") {
		return "", false
	}
	if strings.TrimSpace(lines[len(lines)-1]) != "```" {
		return "", false
	}

	return strings.TrimSpace(strings.Join(lines[1:len(lines)-1], "\n")), true
}

func constructPrompt(baseRefName, baseCommitID, headCommitID string) string {
	prompt := fmt.Sprintf(`
Check this PR and compare it with its target branch %q.
The authoritative PR diff is:
  git diff --no-ext-diff --find-renames --unified=3 %s...%s --
Use this exact commit range rather than guessing the target branch.

This is a non-interactive, read-only review. Do not modify repository files or
request additional permissions. Use only operations available within the
current sandbox; if an operation is denied, try another read-only approach.

Read the README.md file (if any), documentation and check the source code.
Once you have a good understanding of the project, go ahead and review the changes.
Return a valid JSON object with this schema:
{
  "overview": "string",
  "feedback": [
    {
      "level": "p1 | p2 | p3",
      "file": "path relative to the repository root",
      "line": 42,
      "side": "LEFT | RIGHT",
      "text": "string"
    }
  ]
}

Keep "overview" to exactly two short sentences and at most 50 words total:
1. Summarize what the PR changes at a high level.
2. Give the overall assessment with at most one brief reason.
Use "in good shape" when there is no actionable feedback, "can be improved"
when all feedback is p3, and "strongly recommend addressing the feedback" when
any feedback is p1 or p2. Do not list files, functions, implementation steps,
or individual tests, and do not repeat inline feedback details in the overview.

Only report concrete, actionable issues that warrant a code change. Use p1 for
critical correctness or security issues, p2 for significant issues likely to
occur in realistic use, and p3 for minor but concrete issues. Do not report
harmless redundancy, stylistic preferences, optional polish, or intentional
trade-offs without a meaningful negative impact.

Write the overview and feedback text as GitHub-flavored Markdown. Use Markdown
inline-code formatting for identifiers, commands, file paths, flags, and
literal values, but do not over-format ordinary prose.

Keep each feedback item focused on one issue, its strongest concrete impact,
and a suggested fix when useful. Most should be 2–3 sentences and roughly
50–80 words. Prefer a direct causal explanation over a step-by-step execution
trace. Do not catalog every affected caller, exploit variant, or consequence
when one representative example establishes the issue. Aim to stay below 100
words; exceed that only when a shorter explanation would not establish why the
finding is correct.

Every feedback location must be a line displayed in a PR diff hunk. Use RIGHT
and the new-file line number for additions or context lines. Use LEFT and the
old-file line number for deletions. Do not use unchanged lines outside a diff
hunk. Mention feedback without a valid diff location only briefly in the second
overview sentence instead of inventing a location.

Your entire response must be raw JSON, beginning with { and ending with }.
Do not wrap it in a Markdown code fence or include any other text.
`, baseRefName, baseCommitID, headCommitID)

	return strings.TrimSpace(prompt)
}

func postReview(dir string, pr pullRequest, commitID string, review Review, diff pullRequestDiff) {
	ghReview := createGitHubReviewStruct(review, commitID, diff)

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

	cmd.Stdout = io.Discard
	cmd.Stderr = toolOutputWriter(os.Stderr)

	if err := cmd.Run(); err != nil {
		fmt.Printf("gh %s: %v", strings.Join(args, " "), err)
		os.Exit(1)
	}
}

func createGitHubReviewStruct(review Review, commitID string, diff pullRequestDiff) GitHubReview {
	overview := strings.TrimSpace(review.Overview)
	if overview == "" {
		fmt.Println("Review overview cannot be empty")
		os.Exit(1)
	}

	comments := make([]InlineComment, 0, len(review.Feedback))
	unmapped := make([]Feedback, 0)

	for _, feedback := range review.Feedback {
		level := strings.ToLower(strings.TrimSpace(feedback.Level))
		switch level {
		case "p1", "p2", "p3":
		default:
			fmt.Printf("Invalid feedback level in the review %q\n", feedback.Level)
			os.Exit(1)
		}

		text := strings.TrimSpace(feedback.Text)
		if text == "" {
			fmt.Println("Invalid inline review feedback")
			os.Exit(1)
		}

		path, side, reviewable := diff.resolve(feedback.File, feedback.Line, feedback.Side)
		if !reviewable {
			unmapped = append(unmapped, feedback)
			continue
		}

		comments = append(comments, InlineComment{
			Path: path,
			Line: feedback.Line,
			Side: side,
			Body: commentPrefix + fmt.Sprintf(
				"**%s**: %s",
				strings.ToUpper(level),
				text,
			),
		})
	}

	return GitHubReview{
		Body:     commentPrefix + appendUnmappedFeedback(overview, unmapped),
		Event:    "COMMENT",
		CommitID: commitID,
		Comments: comments,
	}
}

func appendUnmappedFeedback(overview string, feedback []Feedback) string {
	if len(feedback) == 0 {
		return overview
	}

	var body strings.Builder
	body.WriteString(overview)
	body.WriteString("\n\n### Additional feedback (not attached to the diff)\n")

	for _, item := range feedback {
		fmt.Fprintf(&body, "- **%s**", strings.ToUpper(strings.TrimSpace(item.Level)))

		path := normalizeFeedbackPath(item.File)
		if path != "" {
			fmt.Fprintf(&body, " `%s", path)
			if item.Line > 0 {
				fmt.Fprintf(&body, ":%d", item.Line)
			}
			body.WriteString("`")

			side := strings.ToUpper(strings.TrimSpace(item.Side))
			if side == diffSideLeft || side == diffSideRight {
				fmt.Fprintf(&body, " (%s)", side)
			}
		}

		fmt.Fprintf(&body, ": %s\n", strings.TrimSpace(item.Text))
	}

	return strings.TrimSpace(body.String())
}
