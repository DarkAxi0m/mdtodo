package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// SendHeartbeat launches a goroutine that checks for the wakatime CLI
// and, if found, runs it in the background with the --write flag to send a heartbeat.
// filename: the path (or name) of the file that triggered the heartbeat
// project:  the name of the project
func SendHeartbeat(filename, project string) {
	go func() {
		if project == "" {
			detectedProject, err := detectProjectName(filename)
			if err != nil {
				log.Printf("Warning: failed to detect project name: %v", err)
			} else {
				project = detectedProject
			}
		}

		cli, err := findWakaTimeCLI()
		if err != nil {
			return
		}

		cmd := exec.Command(
			cli,
			"--write",
			"--entity", filename,
			"--project", project,
			"--plugin", ApplicationName+"/"+ApplicationVersion,
			"--category", "planning",
		)

		if err := cmd.Run(); err != nil {
			log.Printf("Error sending heartbeat via WakaTime CLI: %v\n", err)
		}
	}()
}

// findWakaTimeCLI searches for the WakaTime CLI in the following order:
// 1. "wakatime-cli" in the current PATH
// 2. "wakatime" in the current PATH
// 3. "~/.wakatime/wakatime-cli" (expanding ~ to the user's home directory)
//
// If found, returns the absolute path to the CLI. Otherwise returns an error.
func findWakaTimeCLI() (string, error) {
	if cliPath, err := exec.LookPath("wakatime-cli"); err == nil {
		return cliPath, nil
	}

	if cliPath, err := exec.LookPath("wakatime"); err == nil {
		return cliPath, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory: %w", err)
	}
	fallbackPath := filepath.Join(homeDir, ".wakatime", "wakatime-cli")

	if fi, err := os.Stat(fallbackPath); err == nil && !fi.IsDir() {
		return fallbackPath, nil
	}

	return "", fmt.Errorf("wakatime CLI not found in PATH or at %s", fallbackPath)
}

// detectProjectName walks up from filename's directory until it finds a .git folder.
// If found, returns the name of that directory (which we treat as the project name).
// If not found, returns the immediate parent folder’s name as a fallback.
func detectProjectName(filename string) (string, error) {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(absPath)

	// Walk up the directory tree until root or until .git is found
	for {
		gitPath := filepath.Join(dir, ".git")
		if fi, err := os.Stat(gitPath); err == nil && fi.IsDir() {
			return filepath.Base(dir), nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return filepath.Base(dir), nil
		}
		dir = parent
	}
}
