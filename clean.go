package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func cleanRepositories() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not read home directory")
		os.Exit(1)
	}

	repoDir := filepath.Join(homeDir, ".llyr", "repos")
	if err := clearRepositoryDirectory(repoDir); err != nil {
		fmt.Fprintf(os.Stderr, "Could not clean repository directory at %s: %v\n", repoDir, err)
		os.Exit(1)
	}

	printAction("Removed all cloned repositories")
}

func clearRepositoryDirectory(repoDir string) error {
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		return fmt.Errorf("create repository directory: %w", err)
	}

	entries, err := os.ReadDir(repoDir)
	if err != nil {
		return fmt.Errorf("read repository directory: %w", err)
	}

	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(repoDir, entry.Name())); err != nil {
			return fmt.Errorf("remove %q: %w", entry.Name(), err)
		}
	}

	return nil
}
