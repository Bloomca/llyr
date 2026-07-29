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

func TestReviewPromptRequestsDiffSide(t *testing.T) {
	prompt := constructPrompt()
	for _, expected := range []string{`"side": "LEFT | RIGHT"`, "Use RIGHT", "Use LEFT"} {
		if !strings.Contains(prompt, expected) {
			t.Errorf("prompt does not contain %q", expected)
		}
	}
}
