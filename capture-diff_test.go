package main

import (
	"strings"
	"testing"
)

func TestParsePullRequestDiff(t *testing.T) {
	patch := `diff --git a/old/name.go b/new/name.go
similarity index 90%
rename from old/name.go
rename to new/name.go
--- a/old/name.go
+++ b/new/name.go
@@ -10,4 +10,5 @@ func example() {
 context
-deleted
+replacement
+added
 context
 context
@@ -30 +31 @@ func another() {
-old
+new`

	diff, err := parsePullRequestDiff(strings.NewReader(patch))
	if err != nil {
		t.Fatalf("parsePullRequestDiff() error = %v", err)
	}

	tests := []struct {
		name     string
		path     string
		line     int
		side     string
		wantPath string
		wantSide string
		wantOK   bool
	}{
		{name: "addition", path: "new/name.go", line: 12, side: "RIGHT", wantPath: "new/name.go", wantSide: "RIGHT", wantOK: true},
		{name: "deletion", path: "new/name.go", line: 11, side: "LEFT", wantPath: "new/name.go", wantSide: "LEFT", wantOK: true},
		{name: "old rename path", path: "./old/name.go", line: 30, side: "left", wantPath: "new/name.go", wantSide: "LEFT", wantOK: true},
		{name: "right context", path: "new/name.go", line: 10, side: "RIGHT", wantPath: "new/name.go", wantSide: "RIGHT", wantOK: true},
		{name: "left context", path: "new/name.go", line: 10, side: "LEFT", wantOK: false},
		{name: "outside hunk", path: "new/name.go", line: 20, side: "RIGHT", wantOK: false},
		{name: "wrong side", path: "new/name.go", line: 30, side: "RIGHT", wantOK: false},
		{name: "unknown file", path: "other.go", line: 12, side: "RIGHT", wantOK: false},
		{name: "infer unambiguous side", path: "new/name.go", line: 31, wantPath: "new/name.go", wantSide: "RIGHT", wantOK: true},
		{name: "do not infer ambiguous side", path: "new/name.go", line: 11, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, side, ok := diff.resolve(tt.path, tt.line, tt.side)
			if path != tt.wantPath || side != tt.wantSide || ok != tt.wantOK {
				t.Fatalf("resolve() = (%q, %q, %v), want (%q, %q, %v)", path, side, ok, tt.wantPath, tt.wantSide, tt.wantOK)
			}
		})
	}
}

func TestParsePullRequestDiffHandlesAddedAndDeletedFiles(t *testing.T) {
	patch := `diff --git a/deleted.go b/deleted.go
--- a/deleted.go
+++ /dev/null
@@ -3 +0,0 @@
-deleted
diff --git a/added.go b/added.go
--- /dev/null
+++ b/added.go
@@ -0,0 +7 @@
+added`

	diff, err := parsePullRequestDiff(strings.NewReader(patch))
	if err != nil {
		t.Fatalf("parsePullRequestDiff() error = %v", err)
	}

	if path, side, ok := diff.resolve("deleted.go", 3, "LEFT"); !ok || path != "deleted.go" || side != "LEFT" {
		t.Fatalf("deleted location = (%q, %q, %v)", path, side, ok)
	}
	if path, side, ok := diff.resolve("added.go", 7, "RIGHT"); !ok || path != "added.go" || side != "RIGHT" {
		t.Fatalf("added location = (%q, %q, %v)", path, side, ok)
	}
}
