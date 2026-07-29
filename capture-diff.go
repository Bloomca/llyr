package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	diffSideLeft  = "LEFT"
	diffSideRight = "RIGHT"

	// Pull requests are currently compared with the repository's default
	// branch. The three-dot diff uses the merge base with the captured head.
	pullRequestBaseRef = "origin/HEAD"
)

type diffLocation struct {
	path string
	side string
	line int
}

type pullRequestDiff struct {
	aliases map[string]string
	lines   map[diffLocation]struct{}
}

func newPullRequestDiff() pullRequestDiff {
	return pullRequestDiff{
		aliases: make(map[string]string),
		lines:   make(map[diffLocation]struct{}),
	}
}

func capturePullRequestDiff(dir, headCommit string) (pullRequestDiff, error) {
	headCommit = strings.TrimSpace(headCommit)
	if headCommit == "" {
		return pullRequestDiff{}, fmt.Errorf("head commit cannot be empty")
	}

	comparison := pullRequestBaseRef + "..." + headCommit
	cmd := exec.Command(
		"git",
		"-c", "core.quotePath=true",
		"diff",
		"--no-color",
		"--no-ext-diff",
		"--find-renames",
		"--unified=3",
		comparison,
		"--",
	)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if message := strings.TrimSpace(stderr.String()); message != "" {
			return pullRequestDiff{}, fmt.Errorf("git diff %s: %s", comparison, message)
		}
		return pullRequestDiff{}, fmt.Errorf("git diff %s: %w", comparison, err)
	}

	return parsePullRequestDiff(&stdout)
}

var hunkHeader = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

func parsePullRequestDiff(reader io.Reader) (pullRequestDiff, error) {
	diff := newPullRequestDiff()
	scanner := bufio.NewScanner(reader)
	// Source lines can exceed Scanner's default token limit.
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

	oldPath, path := "", ""
	oldLine, newLine := 0, 0
	inHunk := false

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "diff --git ") {
			oldPath, path = "", ""
			inHunk = false
			continue
		}

		if !inHunk {
			switch {
			case strings.HasPrefix(line, "--- "):
				var err error
				oldPath, err = parseDiffPath(strings.TrimPrefix(line, "--- "), "a/")
				if err != nil {
					return pullRequestDiff{}, err
				}
			case strings.HasPrefix(line, "+++ "):
				newPath, err := parseDiffPath(strings.TrimPrefix(line, "+++ "), "b/")
				if err != nil {
					return pullRequestDiff{}, err
				}

				path = newPath
				if path == "" {
					path = oldPath
				}
				if path != "" {
					diff.aliases[path] = path
					if oldPath != "" {
						diff.aliases[oldPath] = path
					}
				}
			}
		}

		if matches := hunkHeader.FindStringSubmatch(line); matches != nil {
			var err error
			oldLine, err = strconv.Atoi(matches[1])
			if err != nil {
				return pullRequestDiff{}, err
			}
			newLine, err = strconv.Atoi(matches[2])
			if err != nil {
				return pullRequestDiff{}, err
			}
			inHunk = path != ""
			continue
		}

		if !inHunk || line == "" {
			continue
		}

		switch line[0] {
		case ' ':
			// GitHub expects context lines on the right side.
			diff.add(path, diffSideRight, newLine)
			oldLine++
			newLine++
		case '-':
			diff.add(path, diffSideLeft, oldLine)
			oldLine++
		case '+':
			diff.add(path, diffSideRight, newLine)
			newLine++
		case '\\':
			// "No newline at end of file" does not consume a source line.
		default:
			inHunk = false
		}
	}

	if err := scanner.Err(); err != nil {
		return pullRequestDiff{}, err
	}
	return diff, nil
}

func parseDiffPath(path, prefix string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "/dev/null" {
		return "", nil
	}

	if strings.HasPrefix(path, `"`) {
		unquoted, err := strconv.Unquote(path)
		if err != nil {
			return "", fmt.Errorf("parse git diff path %q: %w", path, err)
		}
		path = unquoted
	}

	return normalizeFeedbackPath(strings.TrimPrefix(path, prefix)), nil
}

func (d *pullRequestDiff) add(path, side string, line int) {
	if line > 0 {
		d.lines[diffLocation{path: path, side: side, line: line}] = struct{}{}
	}
}

func (d pullRequestDiff) resolve(path string, line int, side string) (string, string, bool) {
	path = normalizeFeedbackPath(path)
	canonicalPath, exists := d.aliases[path]
	if !exists || line <= 0 {
		return "", "", false
	}

	side = strings.ToUpper(strings.TrimSpace(side))
	if side == "" {
		_, onLeft := d.lines[diffLocation{path: canonicalPath, side: diffSideLeft, line: line}]
		_, onRight := d.lines[diffLocation{path: canonicalPath, side: diffSideRight, line: line}]

		// Older agent responses did not contain a side. Infer it only when the
		// coordinate exists on exactly one side.
		if onLeft == onRight {
			return "", "", false
		}
		if onLeft {
			side = diffSideLeft
		} else {
			side = diffSideRight
		}
	}

	if side != diffSideLeft && side != diffSideRight {
		return "", "", false
	}
	if _, exists := d.lines[diffLocation{path: canonicalPath, side: side, line: line}]; !exists {
		return "", "", false
	}

	return canonicalPath, side, true
}

func normalizeFeedbackPath(path string) string {
	path = strings.TrimSpace(path)
	for strings.HasPrefix(path, "./") {
		path = strings.TrimPrefix(path, "./")
	}
	return path
}
