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

func prepareRepo(link string) (string, pullRequest) {
	pr, err := parsePullRequestLink(link)

	if err != nil {
		fmt.Println("Could not parse PR link: ", err)
		os.Exit(1)
	}

	return preparePullRequest(pr)
}

func preparePullRequest(pr pullRequest) (string, pullRequest) {
	printAction("Fetching pull request details…")
	metadata, err := fetchPullRequestMetadata(pr)
	if err != nil {
		fmt.Println("Could not read pull request metadata: ", err)
		os.Exit(1)
	}
	if metadata.viewerPermission != "ADMIN" {
		fmt.Println("Not the admin of the repository")
		os.Exit(1)
	}

	pr.baseRefName = metadata.baseRefName
	pr.baseCommitID = metadata.baseCommitID
	pr.headCommitID = metadata.headCommitID

	repoDir := clone(pr)
	checkout(repoDir, pr.number)

	return repoDir, pr
}

type pullRequest struct {
	owner        string
	repo         string
	number       int
	baseRefName  string
	baseCommitID string
	headCommitID string
}

func (pr pullRequest) slug() string { return pr.owner + "/" + pr.repo }

func (pr pullRequest) webURL() string {
	return fmt.Sprintf("https://github.com/%s/pull/%d", pr.slug(), pr.number)
}

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

	if u.Scheme != "http" && u.Scheme != "https" {
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

type pullRequestMetadata struct {
	viewerPermission string
	baseRefName      string
	baseCommitID     string
	headCommitID     string
}

const pullRequestMetadataQuery = `
query PullRequestMetadata($owner: String!, $repo: String!, $number: Int!) {
  repository(owner: $owner, name: $repo) {
    viewerPermission
    pullRequest(number: $number) {
      baseRefName
      baseRefOid
      headRefOid
    }
  }
}`

func fetchPullRequestMetadata(pr pullRequest) (pullRequestMetadata, error) {
	output, err := ghCapture(
		"",
		"api", "graphql",
		"-f", "query="+pullRequestMetadataQuery,
		"-F", "owner="+pr.owner,
		"-F", "repo="+pr.repo,
		"-F", "number="+strconv.Itoa(pr.number),
	)
	if err != nil {
		return pullRequestMetadata{}, err
	}

	var payload struct {
		Data struct {
			Repository *struct {
				ViewerPermission string `json:"viewerPermission"`
				PullRequest      *struct {
					BaseRefName  string `json:"baseRefName"`
					BaseCommitID string `json:"baseRefOid"`
					HeadCommitID string `json:"headRefOid"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		return pullRequestMetadata{}, fmt.Errorf("decode pull request metadata: %w", err)
	}
	if len(payload.Errors) > 0 {
		messages := make([]string, 0, len(payload.Errors))
		for _, graphQLError := range payload.Errors {
			messages = append(messages, graphQLError.Message)
		}
		return pullRequestMetadata{}, errors.New(strings.Join(messages, "; "))
	}
	if payload.Data.Repository == nil {
		return pullRequestMetadata{}, errors.New("repository was not found")
	}
	if payload.Data.Repository.PullRequest == nil {
		return pullRequestMetadata{}, errors.New("pull request was not found")
	}

	pull := payload.Data.Repository.PullRequest
	if pull.BaseRefName == "" || pull.BaseCommitID == "" || pull.HeadCommitID == "" {
		return pullRequestMetadata{}, errors.New("pull request metadata is incomplete")
	}

	return pullRequestMetadata{
		viewerPermission: payload.Data.Repository.ViewerPermission,
		baseRefName:      pull.BaseRefName,
		baseCommitID:     pull.BaseCommitID,
		headCommitID:     pull.HeadCommitID,
	}, nil
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
		printAction("Found repository at %s", repoDir)
		return repoDir
	} else {
		printAction("Cloning %s into %s", pr.slug(), repoDir)
		if err = ghStream("", "repo", "clone", pr.slug(), repoDir); err != nil {
			fmt.Printf("cloning %s: %v", pr.slug(), err)
			os.Exit(1)
		}

		return repoDir
	}
}

func checkout(repoDir string, prNumber int) {
	printAction("Checking out pull request #%d", prNumber)
	if err := ghStream(repoDir, "pr", "checkout", "--force", strconv.Itoa(prNumber)); err != nil {
		fmt.Printf("Could not checkout the PR %d at the %s", prNumber, repoDir)
		os.Exit(1)
	}
}

func requireGitHubCLI() error {
	path, err := exec.LookPath("gh")
	if err != nil {
		return errors.New("GitHub CLI (`gh`) was not found in PATH; install it from https://cli.github.com/")
	}

	cmd := exec.Command(path, "auth", "status", "--active", "--hostname", "github.com")
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err != nil {
		message := "GitHub CLI (`gh`) is not authenticated for github.com; run `gh auth login --hostname github.com` and try again"
		if details := strings.TrimSpace(output.String()); details != "" {
			message += ":\n" + details
		}
		return errors.New(message)
	}

	return nil
}

func newGitHubCLI(dir string, args []string) *exec.Cmd {
	cmd := exec.Command("gh", args...)
	cmd.Dir = dir
	return cmd
}

func ghCapture(dir string, args ...string) (string, error) {
	cmd := newGitHubCLI(dir, args)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = toolOutputWriter(os.Stderr)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gh %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// stream the output of the GH command
func ghStream(dir string, args ...string) error {
	cmd := newGitHubCLI(dir, args)
	cmd.Stdout = toolOutputWriter(os.Stdout)
	cmd.Stderr = toolOutputWriter(os.Stderr)
	return cmd.Run()
}
