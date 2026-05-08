package platform

import (
	"fmt"
	"os"
	"path/filepath"
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
