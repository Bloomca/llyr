package main

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type pullRequest struct {
	owner  string
	repo   string
	nunber int
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
		nunber: number,
	}, nil
}
