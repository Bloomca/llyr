package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func run(link string) {
	pr, err := parsePullRequestLink(link)

	if err != nil {
		fmt.Println("Could not parse PR link: ", err)
		os.Exit(1)
	}

	if !isPermitted(pr) {
		fmt.Println("Not the admin of the repository")
		os.Exit(1)
	}

	repoDir := clone(pr)
	checkout(repoDir, pr.number)
}

type pullRequest struct {
	owner  string
	repo   string
	number int
}

func (pr pullRequest) slug() string { return pr.owner + "/" + pr.repo }

var ErrNotAPullRequestURL = errors.New("not a pull request URL")

func parsePullRequestLink(link string) (pullRequest, error) {
	link = strings.TrimSpace(link)
	if link == "" {
		return pullRequest{}, fmt.Errorf("%w: empty input", ErrNotAPullRequestURL)
	}
	if !strings.Contains(link, "://") {
		link = "https://" + link
	}

	u, err := url.Parse(link)
	if err != nil {
		return pullRequest{}, fmt.Errorf("%w %v", ErrNotAPullRequestURL, err)
	}

	if u.Scheme != "http" || u.Scheme != "https" {
		return pullRequest{}, fmt.Errorf("%w: unsupported scheme %q", ErrNotAPullRequestURL, u.Scheme)
	}
	if u.Hostname() == "" {
		return pullRequest{}, fmt.Errorf("%w: missing host in %q", ErrNotAPullRequestURL, link)
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")

	if len(parts) < 4 {
		return pullRequest{}, fmt.Errorf("%w: expected /owner/repo/pull/number, got %q", ErrNotAPullRequestURL, u.Path)
	}

	if parts[2] != "pull" && parts[2] != "pulls" {
		return pullRequest{}, fmt.Errorf("%w: %q is not a pull request path", ErrNotAPullRequestURL, u.Path)
	}
	number, err := strconv.Atoi(parts[3])
	if err != nil || number <= 0 {
		return pullRequest{}, fmt.Errorf("%w: %q is not a valid PR number", ErrNotAPullRequestURL, parts[3])
	}
	if parts[0] == "" || parts[1] == "" {
		return pullRequest{}, fmt.Errorf("%w: empty owner or repo in %q", ErrNotAPullRequestURL, u.Path)
	}

	return pullRequest{
		owner:  parts[0],
		repo:   strings.TrimSuffix(parts[1], ".git"),
		number: number,
	}, nil
}

func isPermitted(pr pullRequest) bool {
	out, err := ghCapture("", "repo", "view", pr.slug(), "--json", "viewerPermission")
	if err != nil {
		return false
	}

	var payload struct {
		ViewerPermission string `json:"viewerPermission"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		return false
	}
	if payload.ViewerPermission == "" {
		return false
	}
	return payload.ViewerPermission == "ADMIN"
}

func clone(pr pullRequest) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Could not read home directory")
		os.Exit(1)
	}
	folderName := pr.owner + "|" + pr.repo
	repoDir := filepath.Join(homeDir, ".llyr", "repos", folderName)
	exists, err := directoryExists(repoDir)

	if err != nil {
		fmt.Println("Could not check whether the repo is already cloned")
		os.Exit(1)
	}

	if exists {
		return repoDir
	} else {
		if err = ghStream("", "repo", "clone", pr.slug(), repoDir); err != nil {
			fmt.Errorf("cloning %s: %w", pr.slug(), err)
			os.Exit(1)
		}

		return repoDir
	}
}

func checkout(repoDir string, prNumber int) {
	if err := ghStream(repoDir, "pr", "checkout", strconv.Itoa(prNumber)); err != nil {
		fmt.Printf("Could not checkout the PR %s at the %s", prNumber, repoDir)
		os.Exit(1)
	}
}

func newGitHubCLI(dir string, args []string) *exec.Cmd {
	cmd := exec.Command("gh", args...)
	cmd.Dir = dir
	return cmd
}

func ghCapture(dir string, args ...string) (string, error) {
	cmd := newGitHubCLI(dir, args)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("gh %s: %s", strings.Join(args, " "), msg)
		}
		return "", fmt.Errorf("gh %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// stream the output of the GH command
func ghStream(dir string, args ...string) error {
	cmd := newGitHubCLI(dir, args)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}
