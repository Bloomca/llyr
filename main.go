package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func main() {
	config := checkConfiguration()

	if len(os.Args) == 1 {
		os.Exit(0)
	}

	prLink := os.Args[1]

	repoDir, pr := prepareRepo(prLink)
	review(config, repoDir, pr)
}

func checkConfiguration() config {
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
		config, err := createConfig()

		if err != nil {
			fmt.Println("Could not create config file: ", err)
			os.Exit(1)
		}

		return config
	} else {
		config, err := parseConfig(configFile)

		if err != nil {
			fmt.Println("Could not parse config file: ", err)
			os.Exit(1)
		}

		return config
	}
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
