package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGitHubReviewIncludesCommitID(t *testing.T) {
	const commitID = "0123456789abcdef"

	review := createGitHubReviewStruct(Review{Overview: "Summary"}, commitID, newPullRequestDiff())
	payload, err := json.Marshal(review)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	if !strings.Contains(string(payload), `"commit_id":"`+commitID+`"`) {
		t.Fatalf("payload does not include commit_id: %s", payload)
	}
}

func TestGitHubReviewPrefixesEveryComment(t *testing.T) {
	const wantPrefix = "> _Posted by [Llŷr](https://github.com/Bloomca/llyr)_\n\n"

	diff := newPullRequestDiff()
	diff.aliases["example.go"] = "example.go"
	diff.add("example.go", diffSideRight, 11)

	got := createGitHubReviewStruct(Review{
		Overview: "Summary",
		Feedback: []Feedback{
			{Level: "p2", File: "example.go", Line: 11, Side: "RIGHT", Text: "suggestion"},
		},
	}, "commit", diff)

	if got.Body != wantPrefix+"Summary" {
		t.Errorf("review body = %q, want %q", got.Body, wantPrefix+"Summary")
	}
	if len(got.Comments) != 1 {
		t.Fatalf("len(Comments) = %d, want 1", len(got.Comments))
	}
	if got.Comments[0].Body != wantPrefix+"**P2**: suggestion" {
		t.Errorf("inline comment body = %q, want %q", got.Comments[0].Body, wantPrefix+"**P2**: suggestion")
	}
}

func TestGitHubReviewValidatesInlineFeedback(t *testing.T) {
	diff := newPullRequestDiff()
	diff.aliases["example.go"] = "example.go"
	diff.add("example.go", diffSideLeft, 10)
	diff.add("example.go", diffSideRight, 11)

	review := Review{
		Overview: "Summary",
		Feedback: []Feedback{
			{Level: "p1", File: "example.go", Line: 10, Side: "LEFT", Text: "deleted line"},
			{Level: "p2", File: "example.go", Line: 11, Side: "RIGHT", Text: "added line"},
			{Level: "p3", File: "example.go", Line: 50, Side: "RIGHT", Text: "outside the diff"},
		},
	}

	got := createGitHubReviewStruct(review, "commit", diff)
	if len(got.Comments) != 2 {
		t.Fatalf("len(Comments) = %d, want 2", len(got.Comments))
	}
	if got.Comments[0].Side != "LEFT" || got.Comments[1].Side != "RIGHT" {
		t.Fatalf("comment sides = %q, %q", got.Comments[0].Side, got.Comments[1].Side)
	}
	if !strings.Contains(got.Body, "outside the diff") {
		t.Fatalf("unmapped feedback was not moved to the review body: %s", got.Body)
	}
}

func TestParseReviewOutputAcceptsRawAndFencedJSON(t *testing.T) {
	const output = `{
  "overview": "Summary"
  ,
  "feedback": [
    {
      "level": "p2",
      "file": "example.go",
      "line": 42,
      "side": "RIGHT",
      "text": "Suggestion"
    }
  ]
}`

	tests := []struct {
		name  string
		input string
	}{
		{name: "raw", input: output},
		{name: "fenced", input: "```json\n" + output + "\n```"},
		{
			name:  "uppercase fence with CRLF",
			input: strings.ReplaceAll("```JSON\n"+output+"\n```", "\n", "\r\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			review, err := parseReviewOutput(tt.input)
			if err != nil {
				t.Fatalf("parseReviewOutput() error = %v", err)
			}
			if review.Overview != "Summary" {
				t.Errorf("Overview = %q, want Summary", review.Overview)
			}
			if len(review.Feedback) != 1 || review.Feedback[0].Text != "Suggestion" {
				t.Errorf("Feedback = %#v", review.Feedback)
			}
		})
	}
}

func TestParseReviewOutputRejectsTextOutsideFence(t *testing.T) {
	_, err := parseReviewOutput("Review:\n```json\n{\"overview\":\"Summary\"}\n```")
	if err == nil {
		t.Fatal("parseReviewOutput() accepted text outside the JSON fence")
	}
}

func TestReviewPromptRequestsDiffSide(t *testing.T) {
	prompt := constructPrompt("release", "base-commit", "head-commit")
	for _, expected := range []string{
		`"side": "LEFT | RIGHT"`,
		"Use RIGHT",
		"Use LEFT",
		"target branch \"release\"",
		"base-commit...head-commit",
		"entire response must be raw JSON",
		"Do not wrap it in a Markdown code fence",
	} {
		if !strings.Contains(prompt, expected) {
			t.Errorf("prompt does not contain %q", expected)
		}
	}
}
