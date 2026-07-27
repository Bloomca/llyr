package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type config struct {
	Version   int    `json:"version"`
	AgentTool string `json:"agentTool"`
}

func (c config) save() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configPath := filepath.Join(homeDir, ".llyr", "config.json")

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

func createConfig() (config, error) {
	config := config{Version: 1, AgentTool: ""}
	config.setTool()
	err := config.save()

	return config, err
}

func parseConfig(fileName string) (config, error) {
	var cfg config
	data, err := os.ReadFile(fileName)

	if err != nil {
		return cfg, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func (c *config) setTool() {
	// check for 3 agents, but the user can have other agents, so
	// it is possible to set up a different CLI option.
	claudeExists := binaryExists("claude")
	codexExists := binaryExists("codex")
	piExists := binaryExists("pi")

	options := []string{}

	if claudeExists {
		options = append(options, "claude")
	}

	if codexExists {
		options = append(options, "codex")
	}

	if piExists {
		options = append(options, "pi")
	}

	options = append(options, "Write a CLI app manually")

	option, err := selectOption(options)

	if err != nil {
		fmt.Println("Could not select anything", err)
		os.Exit(1)
	}

	// last choice is always select your own
	if option == len(options)-1 {
		// Wait for the line input
	} else {
		c.AgentTool = getToolOptions(options[option])
	}

	c.save()
}

func getToolOptions(tool string) string {
	if tool == "claude" {
		return "claude -p"
	}

	if tool == "codex" {
		return "codex exec"
	}

	if tool == "pi" {
		return "pi -p"
	}

	return tool
}

func binaryExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
