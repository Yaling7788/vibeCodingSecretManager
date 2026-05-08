package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"

	"github.com/YOUR_GITHUB_USERNAME/vibeCodingSecretManager/internal/config"
	"github.com/YOUR_GITHUB_USERNAME/vibeCodingSecretManager/internal/keepassxc"
	"github.com/YOUR_GITHUB_USERNAME/vibeCodingSecretManager/internal/platform"
	"github.com/YOUR_GITHUB_USERNAME/vibeCodingSecretManager/internal/redaction"
	"github.com/YOUR_GITHUB_USERNAME/vibeCodingSecretManager/internal/runner"
	"golang.org/x/term"
)

const defaultConfigPath = "~/.config/vibeCodingSecretManager/config.yaml"

func main() {
	if code := run(os.Args[1:]); code != 0 {
		os.Exit(code)
	}
}

func run(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 2
	}

	cfgPath := defaultConfigPath
	if args[0] == "--config" || args[0] == "-c" {
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "error: --config requires a path")
			return 2
		}
		cfgPath = args[1]
		args = args[2:]
	}

	switch args[0] {
	case "run":
		return runCommand(cfgPath, args[1:])
	case "check":
		return checkCommand(cfgPath, args[1:])
	case "list":
		return listCommand(cfgPath, args[1:])
	case "init":
		return initCommand(cfgPath)
	case "help", "-h", "--help":
		printUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command %q\n\n", args[0])
		printUsage()
		return 2
	}
}

func runCommand(cfgPath string, args []string) int {
	project, environment, cmdArgs, ok := parseProjectEnvCommand(args)
	if !ok {
		fmt.Fprintln(os.Stderr, "usage: vibeCodingSecretManager run <project> <environment> -- <command...>")
		return 2
	}

	cfg, selection, err := loadSelection(cfgPath, project, environment)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	password, err := promptPassword()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	defer zeroBytes(password)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	vault, err := keepassxc.New(cfg.Vault)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	env, values, err := buildChildEnv(ctx, vault, selection.Environment.Secrets, string(password))
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", redaction.Redact(err.Error(), values))
		return 1
	}
	defer zeroStrings(values)

	exitCode, err := runner.Run(ctx, runner.Invocation{
		Command: cmdArgs,
		Dir:     selection.Project.Root,
		Env:     env,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", redaction.Redact(err.Error(), values))
		return exitCodeOrOne(exitCode)
	}

	return exitCode
}

func checkCommand(cfgPath string, args []string) int {
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: vibeCodingSecretManager check <project> <environment>")
		return 2
	}

	cfg, selection, err := loadSelection(cfgPath, args[0], args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	if err := checkPaths(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	password, err := promptPassword()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	defer zeroBytes(password)

	vault, err := keepassxc.New(cfg.Vault)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	keys := sortedSecretNames(selection.Environment.Secrets)
	for _, envName := range keys {
		entry := selection.Environment.Secrets[envName]
		if _, err := vault.Password(ctx, entry, string(password)); err != nil {
			fmt.Fprintf(os.Stderr, "error: configured entry for %s could not be read: %s\n", envName, redaction.Redact(err.Error(), nil))
			return 1
		}
	}

	fmt.Printf("Project: %s\nEnvironment: %s\n\n", args[0], args[1])
	fmt.Println("Check passed.")
	fmt.Printf("Configured variables: %d\n", len(selection.Environment.Secrets))
	return 0
}

func listCommand(cfgPath string, args []string) int {
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: vibeCodingSecretManager list <project> <environment>")
		return 2
	}

	_, selection, err := loadSelection(cfgPath, args[0], args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	fmt.Printf("Project: %s\nEnvironment: %s\n\n", args[0], args[1])
	fmt.Println("Configured variables:")
	for _, envName := range sortedSecretNames(selection.Environment.Secrets) {
		fmt.Printf("  %s -> %s\n", envName, selection.Environment.Secrets[envName])
	}
	return 0
}

func initCommand(cfgPath string) int {
	path, err := platform.ExpandPath(cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(os.Stderr, "error: config already exists at %s\n", path)
		return 1
	} else if !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	if err := os.WriteFile(path, []byte(config.StarterConfig), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	fmt.Println("Created starter config:", path)
	return 0
}

func loadSelection(cfgPath, project, environment string) (*config.Config, config.Selection, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, config.Selection{}, err
	}

	selection, err := cfg.Select(project, environment)
	if err != nil {
		return nil, config.Selection{}, err
	}
	return cfg, selection, nil
}

func parseProjectEnvCommand(args []string) (string, string, []string, bool) {
	if len(args) < 4 {
		return "", "", nil, false
	}
	if args[2] != "--" || len(args[3:]) == 0 {
		return "", "", nil, false
	}
	return args[0], args[1], args[3:], true
}

func buildChildEnv(ctx context.Context, vault *keepassxc.Client, secrets map[string]string, password string) ([]string, []string, error) {
	childEnv := os.Environ()
	values := make([]string, 0, len(secrets))

	for _, envName := range sortedSecretNames(secrets) {
		entry := secrets[envName]
		secret, err := vault.Password(ctx, entry, password)
		if err != nil {
			return nil, values, fmt.Errorf("failed to retrieve configured secret %s: %w", envName, err)
		}
		values = append(values, secret)
		childEnv = append(childEnv, envName+"="+secret)
	}

	return childEnv, values, nil
}

func checkPaths(cfg *config.Config) error {
	if _, err := os.Stat(cfg.Vault.Database); err != nil {
		return fmt.Errorf("KeePassXC database is not accessible: %w", err)
	}

	if cfg.Vault.KeyFile != "" {
		if _, err := os.Stat(cfg.Vault.KeyFile); err != nil {
			return fmt.Errorf("KeePassXC key file is not accessible: %w", err)
		}
	}

	cli := cfg.Vault.CLIPath
	if cli == "" || cli == "auto" {
		cli = "keepassxc-cli"
	}
	if _, err := exec.LookPath(cli); err != nil {
		return fmt.Errorf("%s was not found on PATH", cli)
	}

	return nil
}

func promptPassword() ([]byte, error) {
	fmt.Fprint(os.Stderr, "KeePassXC master password: ")
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("failed to read hidden password: %w", err)
	}
	return password, nil
}

func sortedSecretNames(secrets map[string]string) []string {
	keys := make([]string, 0, len(secrets))
	for key := range secrets {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func zeroBytes(value []byte) {
	for i := range value {
		value[i] = 0
	}
}

func zeroStrings(values []string) {
	for i := range values {
		values[i] = strings.Repeat("x", len(values[i]))
	}
}

func exitCodeOrOne(code int) int {
	if code == 0 {
		return 1
	}
	return code
}

func printUsage() {
	name := "vibeCodingSecretManager"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	fmt.Fprintf(os.Stderr, `%s safely injects KeePassXC secrets into local development commands.

Usage:
  %s [--config path] run <project> <environment> -- <command...>
  %s [--config path] check <project> <environment>
  %s [--config path] list <project> <environment>
  %s [--config path] init

Examples:
  %s run sample-webapp dev -- npm run dev
  %s run sample-webapp dev -- docker compose up
  %s check sample-webapp dev
`, name, name, name, name, name, name, name, name)
}
