package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func main() {
	checkConfiguration()
}

func checkConfiguration() {
	homeDir, err := os.UserHomeDir()

	if err != nil {
		fmt.Println("Could not read home directory (used to save repos and config)")
		os.Exit(1)
	}

	dir := filepath.Join(homeDir, ".llyr")

	exists, err := directoryExists(dir)

	if err != nil {
		fmt.Println("Could not read app directory (used to save repos and config)")
		os.Exit(1)
	}

	if !exists {
		if err = os.Mkdir(dir, 0o755); err != nil {
			fmt.Printf("Could not create app directory at %v", dir)
			os.Exit(1)
		}

		repoDir := filepath.Join(dir, "repos")
		if err = os.Mkdir(repoDir, 0o755); err != nil {
			fmt.Printf("Could not create repo directory at %v", repoDir)
			os.Exit(1)
		}
	} else {
		repoDir := filepath.Join(dir, "repos")
		exists, err = directoryExists(repoDir)

		if err != nil {
			fmt.Println("Could not read repo directory")
			os.Exit(1)
		}

		if !exists {
			if err = os.Mkdir(repoDir, 0o755); err != nil {
				fmt.Printf("Could not create repo directory at %v", repoDir)
				os.Exit(1)
			}
		}
	}

	configFile := filepath.Join(dir, "config.json")

	exists, err = fileExists(configFile)

	if err != nil {
		fmt.Printf("Could not read the config file at %v", configFile)
		os.Exit(1)
	}

	if !exists {
		// create a config file
	}

	// 1. check that the folder ~home/.llyr exists
	// 2. if not, create it
	// 3. check the configuration file (tool + gh)
	// 4. if gh is not installed, exit
	// 5. allow user to select the review agent
}

func directoryExists(dir string) (bool, error) {
	info, err := os.Stat(dir)
	if err == nil {
		return info.IsDir(), nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}

	return false, err
}

func fileExists(name string) (bool, error) {
	info, err := os.Stat(name)
	if err == nil {
		return info.Mode().IsRegular(), nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}

	return false, err
}
