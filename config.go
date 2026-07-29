package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	cfg := config{Version: 1}
	if err := cfg.setTool(); err != nil {
		return cfg, err
	}

	return cfg, cfg.save()
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

func (c *config) setTool() error {
	// Check for 3 agents, but the user can have other agents, so
	// it is possible to set up a different CLI option.
	options := []string{}

	if binaryExists("claude") {
		options = append(options, "claude")
	}

	if binaryExists("codex") {
		options = append(options, "codex")
	}

	if binaryExists("pi") {
		options = append(options, "pi")
	}

	options = append(options, "Write a CLI app manually")

	option, err := selectOption(options)
	if err != nil {
		return fmt.Errorf("select a tool: %w", err)
	}

	// The last choice is always select your own.
	if option == len(options)-1 {
		tool, err := readAgentTool(os.Stdin, os.Stdout)
		if err != nil {
			return err
		}
		c.AgentTool = tool
	} else {
		c.AgentTool = getToolOptions(options[option])
	}

	return nil
}

func readAgentTool(in io.Reader, out io.Writer) (string, error) {
	fmt.Fprint(out, "Enter the CLI command and any arguments: ")

	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read CLI command: %w", err)
	}

	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	args, err := splitArguments(line)
	if err != nil {
		return "", fmt.Errorf("invalid CLI command: %w", err)
	}

	if _, err := exec.LookPath(args[0]); err != nil {
		return "", fmt.Errorf("look up CLI tool %q: %w", args[0], err)
	}

	return line, nil
}

func getToolOptions(tool string) string {
	if tool == "claude" {
		return "claude -p --permission-mode auto"
	}

	if tool == "codex" {
		return "codex -a never -s read-only exec"
	}

	if tool == "pi" {
		return "pi --no-approve --tools read,bash,grep,find,ls -p"
	}

	return tool
}

func binaryExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
