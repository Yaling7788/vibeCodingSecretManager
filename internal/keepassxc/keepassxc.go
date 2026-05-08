package keepassxc

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/Yaling7788/vibeCodingSecretManager/internal/config"
	"github.com/Yaling7788/vibeCodingSecretManager/internal/redaction"
)

type Client struct {
	database string
	keyFile  string
	cliPath  string
}

func New(vault config.Vault) (*Client, error) {
	cliPath := vault.CLIPath
	if cliPath == "" || cliPath == "auto" {
		cliPath = "keepassxc-cli"
	}
	if _, err := exec.LookPath(cliPath); err != nil {
		return nil, fmt.Errorf("%s was not found on PATH", cliPath)
	}

	return &Client{
		database: vault.Database,
		keyFile:  vault.KeyFile,
		cliPath:  cliPath,
	}, nil
}

func (c *Client) Password(ctx context.Context, entryPath, masterPassword string) (string, error) {
	args := []string{"show", "-q", "-a", "Password"}
	if c.keyFile != "" {
		args = append(args, "-k", c.keyFile)
	}
	args = append(args, c.database, entryPath)

	cmd := exec.CommandContext(ctx, c.cliPath, args...)
	cmd.Stdin = strings.NewReader(masterPassword + "\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		safeErr := strings.TrimSpace(stderr.String())
		if safeErr == "" {
			safeErr = err.Error()
		}
		return "", fmt.Errorf("%s", redaction.Redact(safeErr, []string{masterPassword}))
	}

	return strings.TrimRight(stdout.String(), "\r\n"), nil
}
