package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Yaling7788/vibeCodingSecretManager/internal/api"
	"github.com/Yaling7788/vibeCodingSecretManager/internal/ipc"
	"github.com/Yaling7788/vibeCodingSecretManager/internal/model"
	"github.com/Yaling7788/vibeCodingSecretManager/internal/platform"
	"github.com/Yaling7788/vibeCodingSecretManager/internal/securecrypto"
	"github.com/Yaling7788/vibeCodingSecretManager/internal/vault"
	"golang.org/x/term"
)

type options struct{ vault, endpoint string }

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	paths, err := platform.DefaultPaths()
	if err != nil {
		return printError(err)
	}
	global := flag.NewFlagSet("vcsm", flag.ContinueOnError)
	global.SetOutput(os.Stderr)
	vaultPath := global.String("vault", paths.Vault, "encrypted vault database")
	endpoint := global.String("endpoint", paths.Endpoint, "local broker endpoint")
	if err := global.Parse(args); err != nil {
		return 2
	}
	args = global.Args()
	if len(args) == 0 {
		usage()
		return 2
	}
	opts := options{vault: *vaultPath, endpoint: *endpoint}
	switch args[0] {
	case "init":
		return initVault(opts, args[1:])
	case "status":
		return callAndPrint(opts, "status", nil)
	case "unlock":
		return unlock(opts)
	case "lock":
		return callAndPrint(opts, "lock", nil)
	case "secret":
		return secretCommand(opts, args[1:])
	case "action":
		return actionCommand(opts, args[1:])
	case "run":
		return runCommand(opts, args[1:])
	case "audit":
		return auditCommand(opts, args[1:])
	case "help", "-h", "--help":
		usage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", args[0])
		usage()
		return 2
	}
}

func initVault(opts options, args []string) int {
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "usage: vcsm init")
		return 2
	}
	password, err := promptPassword("New VCSM master password: ")
	if err != nil {
		return printError(err)
	}
	defer securecrypto.Zero(password)
	confirmation, err := promptPassword("Confirm master password: ")
	if err != nil {
		return printError(err)
	}
	defer securecrypto.Zero(confirmation)
	if string(password) != string(confirmation) {
		return printError(errors.New("passwords do not match"))
	}
	if len(password) < 15 {
		return printError(errors.New("master password must be at least 15 characters"))
	}
	if len(password) > 256 {
		return printError(errors.New("master password must be at most 256 characters"))
	}
	store, err := vault.Initialize(context.Background(), opts.vault, password, securecrypto.DefaultKDFParams())
	if err != nil {
		return printError(err)
	}
	id := store.VaultID()
	_ = store.Close()
	fmt.Printf("Initialized protected vault %s at %s\nRestart or start vcsm-broker, then run vcsm unlock.\n", id, opts.vault)
	return 0
}

func unlock(opts options) int {
	password, err := promptPassword("VCSM master password: ")
	if err != nil {
		return printError(err)
	}
	defer securecrypto.Zero(password)
	return callAndPrint(opts, "unlock", api.UnlockRequest{Password: password})
}

func secretCommand(opts options, args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: vcsm secret {create|rotate|revoke|list} <project> <environment> [name] [--profile profile]")
		return 2
	}
	verb, scope := args[0], model.Scope{Project: args[1], Environment: args[2]}
	if verb == "list" {
		if len(args) != 3 {
			return 2
		}
		return callAndPrint(opts, "secret.list", api.ScopeRequest{Scope: scope})
	}
	if len(args) < 4 {
		return 2
	}
	request := api.SecretRequest{Scope: scope, Name: args[3], Profile: "default"}
	if len(args) == 6 && args[4] == "--profile" {
		request.Profile = args[5]
	} else if len(args) != 4 {
		return 2
	}
	switch verb {
	case "create", "rotate", "revoke":
		return callAndPrint(opts, "secret."+verb, request)
	default:
		return 2
	}
}

func actionCommand(opts options, args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: vcsm action list <project> <environment> | vcsm action configure ...")
		return 2
	}
	if args[0] == "list" {
		if len(args) != 3 {
			return 2
		}
		return callAndPrint(opts, "action.list", api.ScopeRequest{Scope: model.Scope{Project: args[1], Environment: args[2]}})
	}
	if args[0] != "configure" || len(args) < 6 {
		return 2
	}
	action := model.Action{Project: args[1], Environment: args[2], Name: args[3], Secrets: map[string]string{}, OutputPolicy: "metadata"}
	index := 4
	for index < len(args) && args[index] != "--" {
		switch args[index] {
		case "--cwd":
			if index+1 >= len(args) {
				return 2
			}
			action.Directory = args[index+1]
			index += 2
		case "--secret":
			if index+1 >= len(args) {
				return 2
			}
			envName, secretName, ok := strings.Cut(args[index+1], "=")
			if !ok {
				return 2
			}
			action.Secrets[envName] = secretName
			index += 2
		case "--output":
			if index+1 >= len(args) {
				return 2
			}
			action.OutputPolicy = args[index+1]
			index += 2
		default:
			return 2
		}
	}
	if index >= len(args) || args[index] != "--" || index+1 >= len(args) {
		return 2
	}
	action.Executable = args[index+1]
	action.Arguments = append([]string(nil), args[index+2:]...)
	password, err := promptPassword("Approve action with master password: ")
	if err != nil {
		return printError(err)
	}
	defer securecrypto.Zero(password)
	return callAndPrint(opts, "action.configure", api.ConfigureActionRequest{Action: action, MasterPassword: password})
}

func runCommand(opts options, args []string) int {
	if len(args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: vcsm run <project> <environment> <approved-action>")
		return 2
	}
	request := api.ActionRequest{Scope: model.Scope{Project: args[0], Environment: args[1]}, Name: args[2]}
	exitCode := 1
	err := call(opts, "run", request, func(response api.Response) error {
		if response.Event == "stdout" || response.Event == "stderr" {
			var output string
			if err := json.Unmarshal(response.Data, &output); err != nil {
				return err
			}
			if response.Event == "stdout" {
				_, _ = fmt.Fprint(os.Stdout, output)
			} else {
				_, _ = fmt.Fprint(os.Stderr, output)
			}
			return nil
		}
		if response.Final {
			var result api.RunResult
			_ = json.Unmarshal(response.Data, &result)
			exitCode = result.ExitCode
			if !response.OK {
				return errors.New(response.Error)
			}
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		if exitCode == 0 {
			return 1
		}
	}
	return exitCode
}

func auditCommand(opts options, args []string) int {
	limit := 100
	if len(args) == 1 {
		parsed, err := strconv.Atoi(args[0])
		if err != nil {
			return 2
		}
		limit = parsed
	} else if len(args) > 1 {
		return 2
	}
	return callAndPrint(opts, "audit.list", api.AuditRequest{Limit: limit})
}

func callAndPrint(opts options, operation string, payload any) int {
	return printCallResult(call(opts, operation, payload, func(response api.Response) error {
		if !response.OK {
			return errors.New(response.Error)
		}
		if response.Final && len(response.Data) > 0 {
			var formatted any
			if json.Unmarshal(response.Data, &formatted) == nil {
				output, _ := json.MarshalIndent(formatted, "", "  ")
				fmt.Println(string(output))
			}
		}
		return nil
	}))
}

func call(opts options, operation string, payload any, handler func(api.Response) error) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	idBytes := make([]byte, 8)
	_, _ = rand.Read(idBytes)
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()
	return ipc.Call(ctx, opts.endpoint, api.Request{ID: hex.EncodeToString(idBytes), Operation: operation, Payload: encoded}, handler)
}

func promptPassword(label string) ([]byte, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil, errors.New("master password requires a local interactive terminal; it cannot be piped, passed as an argument, or read from an environment variable")
	}
	fmt.Fprint(os.Stderr, label)
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return nil, err
	}
	return password, nil
}

func printCallResult(err error) int {
	if err != nil {
		return printError(err)
	}
	return 0
}
func printError(err error) int { fmt.Fprintln(os.Stderr, "error:", err); return 1 }

func usage() {
	fmt.Fprintln(os.Stderr, `VCSM protected-mode CLI

Usage:
  vcsm init
  vcsm status | unlock | lock
  vcsm secret create|rotate <project> <environment> <name> [--profile default|alnum|password|database]
  vcsm secret revoke <project> <environment> <name>
  vcsm secret list <project> <environment>
  vcsm action configure <project> <environment> <name> [--cwd dir] [--secret ENV=secret] [--output metadata|redacted] -- executable [args...]
  vcsm action list <project> <environment>
  vcsm run <project> <environment> <approved-action>
  vcsm audit [limit]

Global options --vault and --endpoint must appear before the command.`)
}
