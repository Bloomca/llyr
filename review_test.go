package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGitHubReviewIncludesCommitID(t *testing.T) {
	const commitID = "0123456789abcdef"

	review := createGitHubReviewStruct(Review{Overview: "Summary"}, commitID)
	payload, err := json.Marshal(review)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	if !strings.Contains(string(payload), `"commit_id":"`+commitID+`"`) {
		t.Fatalf("payload does not include commit_id: %s", payload)
	}
}
