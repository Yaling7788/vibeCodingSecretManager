package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to resolve home directory: %w", err)
		}
		return home, nil
	}

	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to resolve home directory: %w", err)
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}

	return filepath.Clean(os.ExpandEnv(path)), nil
}

type Paths struct {
	DataDir  string
	Vault    string
	Endpoint string
}

func DefaultPaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}
	var data, endpoint string
	switch runtime.GOOS {
	case "darwin":
		data = filepath.Join(home, "Library", "Application Support", "VCSM")
		endpoint = filepath.Join(home, "Library", "Caches", "VCSM", "broker.sock")
	case "windows":
		data = os.Getenv("LOCALAPPDATA")
		if data == "" {
			data = filepath.Join(home, "AppData", "Local")
		}
		data = filepath.Join(data, "VCSM")
		endpoint = `\\.\pipe\vcsm-broker`
	default:
		data = os.Getenv("XDG_STATE_HOME")
		if data == "" {
			data = filepath.Join(home, ".local", "state")
		}
		data = filepath.Join(data, "vcsm")
		runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
		if runtimeDir == "" {
			runtimeDir = data
		}
		endpoint = filepath.Join(runtimeDir, "vcsm", "broker.sock")
	}
	return Paths{DataDir: data, Vault: filepath.Join(data, "vault.db"), Endpoint: endpoint}, nil
}
