package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	ApplicationName    = "mdtodo"
	ApplicationVersion = "0.0.1"
	BindingConfig      = "keybinding.json"
)

// move to shared some stage... maybe
func getUserConfigPath(filename string) (string, error) {
	var configDir string

	switch runtime.GOOS {
	case "windows":
		configDir = os.Getenv("APPDATA")
	case "darwin", "linux":
		configDir = os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			configDir = filepath.Join(home, ".config")
		}
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	return filepath.Join(configDir, ApplicationName, filename), nil
}
