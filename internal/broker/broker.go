package broker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Yaling7788/vibeCodingSecretManager/internal/api"
	"github.com/Yaling7788/vibeCodingSecretManager/internal/generator"
	"github.com/Yaling7788/vibeCodingSecretManager/internal/model"
	"github.com/Yaling7788/vibeCodingSecretManager/internal/power"
	"github.com/Yaling7788/vibeCodingSecretManager/internal/redaction"
	"github.com/Yaling7788/vibeCodingSecretManager/internal/runner"
	"github.com/Yaling7788/vibeCodingSecretManager/internal/securecrypto"
	"github.com/Yaling7788/vibeCodingSecretManager/internal/vault"
)

var safeName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.-]{0,127}$`)
var safeEnvName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type Broker struct {
	mu        sync.RWMutex
	vaultPath string
	store     *vault.Store
	startedAt time.Time
	power     power.Manager
}

func New(vaultPath string) (*Broker, error) {
	b := &Broker{vaultPath: vaultPath, startedAt: time.Now().UTC()}
	store, err := vault.Open(vaultPath)
	if err == nil {
		b.store = store
		return b, nil
	}
	if errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "no such file") {
		return b, nil
	}
	return nil, err
}

func (b *Broker) Close() error {
	b.power.Stop()
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.store != nil {
		return b.store.Close()
	}
	return nil
}

func (b *Broker) Handle(ctx context.Context, request api.Request, send func(api.Response) error) {
	fail := func(err error) { _ = send(api.Response{OK: false, Error: publicError(err), Final: true}) }
	done := func(value any) {
		encoded, err := json.Marshal(value)
		if err != nil {
			fail(err)
			return
		}
		_ = send(api.Response{OK: true, Data: encoded, Final: true})
	}

	switch request.Operation {
	case "status":
		done(b.status())
	case "unlock":
		var payload api.UnlockRequest
		if err := json.Unmarshal(request.Payload, &payload); err != nil {
			fail(err)
			return
		}
		defer securecrypto.Zero(payload.Password)
		store, err := b.getStore()
		if err == nil {
			err = store.Unlock(ctx, payload.Password)
		}
		if err == nil {
			err = b.power.Start()
		}
		if err != nil {
			fail(err)
			return
		}
		done(b.status())
	case "lock":
		store, err := b.getStore()
		if err != nil {
			fail(err)
			return
		}
		store.Lock()
		b.power.Stop()
		done(b.status())
	case "secret.create", "secret.rotate":
		var payload api.SecretRequest
		if err := json.Unmarshal(request.Payload, &payload); err != nil {
			fail(err)
			return
		}
		if err := validateSecretRequest(payload); err != nil {
			fail(err)
			return
		}
		value, profile, err := generator.Generate(payload.Profile)
		if err != nil {
			fail(err)
			return
		}
		defer securecrypto.Zero(value)
		store, err := b.getStore()
		if err != nil {
			fail(err)
			return
		}
		var metadata model.SecretMetadata
		if request.Operation == "secret.create" {
			metadata, err = store.CreateSecret(ctx, model.SecretMetadata{Project: payload.Scope.Project, Environment: payload.Scope.Environment, Name: payload.Name, Profile: profile.Name, StrengthBits: profile.StrengthBits}, value)
		} else {
			metadata, err = store.RotateSecret(ctx, payload.Scope, payload.Name, profile.Name, profile.StrengthBits, value)
		}
		if err != nil {
			fail(err)
			return
		}
		done(metadata)
	case "secret.revoke":
		var payload api.SecretRequest
		if err := json.Unmarshal(request.Payload, &payload); err != nil {
			fail(err)
			return
		}
		store, err := b.getStore()
		if err != nil {
			fail(err)
			return
		}
		metadata, err := store.RevokeSecret(ctx, payload.Scope, payload.Name)
		if err != nil {
			fail(err)
			return
		}
		done(metadata)
	case "secret.list":
		var payload api.ScopeRequest
		if err := json.Unmarshal(request.Payload, &payload); err != nil {
			fail(err)
			return
		}
		store, err := b.getStore()
		if err != nil {
			fail(err)
			return
		}
		values, err := store.ListSecrets(ctx, payload.Scope)
		if err != nil {
			fail(err)
			return
		}
		done(values)
	case "action.configure":
		var payload api.ConfigureActionRequest
		if err := json.Unmarshal(request.Payload, &payload); err != nil {
			fail(err)
			return
		}
		defer securecrypto.Zero(payload.MasterPassword)
		if payload.Action.OutputPolicy == "" {
			payload.Action.OutputPolicy = "metadata"
		}
		if err := validateAction(payload.Action); err != nil {
			fail(err)
			return
		}
		store, err := b.getStore()
		if err != nil {
			fail(err)
			return
		}
		if err := store.VerifyPassword(ctx, payload.MasterPassword); err != nil {
			fail(err)
			return
		}
		action, err := store.PutAction(ctx, payload.Action)
		if err != nil {
			fail(err)
			return
		}
		done(action)
	case "action.list":
		var payload api.ScopeRequest
		if err := json.Unmarshal(request.Payload, &payload); err != nil {
			fail(err)
			return
		}
		store, err := b.getStore()
		if err != nil {
			fail(err)
			return
		}
		actions, err := store.ListActions(ctx, payload.Scope)
		if err != nil {
			fail(err)
			return
		}
		for index := range actions {
			actions[index].Secrets = copySecretNames(actions[index].Secrets)
		}
		done(actions)
	case "run":
		var payload api.ActionRequest
		if err := json.Unmarshal(request.Payload, &payload); err != nil {
			fail(err)
			return
		}
		if err := b.runAction(ctx, payload, send); err != nil {
			fail(err)
			return
		}
	case "audit.list":
		var payload api.AuditRequest
		if err := json.Unmarshal(request.Payload, &payload); err != nil {
			fail(err)
			return
		}
		store, err := b.getStore()
		if err != nil {
			fail(err)
			return
		}
		events, err := store.ListAudit(ctx, payload.Limit)
		if err != nil {
			fail(err)
			return
		}
		done(events)
	default:
		fail(fmt.Errorf("unknown operation %q", request.Operation))
	}
}

func (b *Broker) runAction(ctx context.Context, payload api.ActionRequest, send func(api.Response) error) error {
	store, err := b.getStore()
	if err != nil {
		return err
	}
	action, err := store.GetAction(ctx, payload.Scope, payload.Name)
	if err != nil {
		return err
	}
	values, err := store.ResolveSecrets(ctx, payload.Scope, action.Secrets)
	if err != nil {
		return err
	}
	defer zeroValues(values)
	secretValues := make([][]byte, 0, len(values))
	environment := baseEnvironment()
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		secretValues = append(secretValues, values[key])
		environment = append(environment, key+"="+string(values[key]))
	}
	stdoutTarget, stderrTarget := io.Writer(io.Discard), io.Writer(io.Discard)
	if action.OutputPolicy == "redacted" {
		stdoutTarget = eventWriter{event: "stdout", send: send}
		stderrTarget = eventWriter{event: "stderr", send: send}
	}
	stdout := redaction.NewWriter(stdoutTarget, secretValues)
	stderr := redaction.NewWriter(stderrTarget, secretValues)
	code, runErr := runner.Run(ctx, runner.Invocation{Command: append([]string{action.Executable}, action.Arguments...), Dir: action.Directory, Env: environment, Stdout: stdout, Stderr: stderr})
	_ = stdout.Close()
	_ = stderr.Close()
	outcome := "success"
	if runErr != nil {
		outcome = "failure"
	}
	_ = store.RecordAudit(ctx, model.AuditEvent{Actor: "agent", Operation: "action.run", ObjectID: action.ID, Outcome: outcome, Detail: fmt.Sprintf("exit_code=%d", code)})
	result, _ := json.Marshal(api.RunResult{Action: action.Name, ExitCode: code})
	if err := send(api.Response{OK: runErr == nil, Data: result, Error: errorText(runErr), Final: true}); err != nil {
		return err
	}
	return nil
}

type eventWriter struct {
	event string
	send  func(api.Response) error
}

func (w eventWriter) Write(p []byte) (int, error) {
	data, _ := json.Marshal(string(p))
	if err := w.send(api.Response{OK: true, Event: w.event, Data: data}); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (b *Broker) getStore() (*vault.Store, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.store == nil {
		store, err := vault.Open(b.vaultPath)
		if err != nil {
			return nil, fmt.Errorf("vault is not initialized; run vcsm init: %w", err)
		}
		b.store = store
	}
	return b.store, nil
}

func (b *Broker) status() api.Status {
	_, _ = b.getStore()
	b.mu.RLock()
	defer b.mu.RUnlock()
	status := api.Status{Initialized: b.store != nil, StartedAt: b.startedAt, UnlockMode: "until-broker-restart", PowerHold: b.power.Active()}
	if b.store != nil {
		status.Unlocked = b.store.IsUnlocked()
		status.VaultID = b.store.VaultID()
	}
	return status
}

func validateSecretRequest(request api.SecretRequest) error {
	if !safeName.MatchString(request.Scope.Project) || !safeName.MatchString(request.Scope.Environment) || !safeName.MatchString(request.Name) {
		return fmt.Errorf("project, environment, and secret name must use letters, digits, dot, dash, or underscore")
	}
	_, err := generator.Get(request.Profile)
	return err
}

func validateAction(action model.Action) error {
	if !safeName.MatchString(action.Project) || !safeName.MatchString(action.Environment) || !safeName.MatchString(action.Name) {
		return fmt.Errorf("invalid action scope or name")
	}
	if action.Executable == "" {
		return fmt.Errorf("action executable is required")
	}
	if action.OutputPolicy != "metadata" && action.OutputPolicy != "redacted" {
		return fmt.Errorf("output policy must be metadata or redacted")
	}
	for envName, secretName := range action.Secrets {
		if !safeEnvName.MatchString(envName) || !safeName.MatchString(secretName) {
			return fmt.Errorf("invalid secret mapping")
		}
	}
	return nil
}

func copySecretNames(input map[string]string) map[string]string {
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
func zeroValues(values map[string][]byte) {
	for key, value := range values {
		securecrypto.Zero(value)
		delete(values, key)
	}
}
func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
func publicError(err error) string {
	if errors.Is(err, vault.ErrInvalidPassword) {
		return vault.ErrInvalidPassword.Error()
	}
	return err.Error()
}

func baseEnvironment() []string {
	allowed := map[string]bool{"PATH": true, "HOME": true, "USER": true, "LOGNAME": true, "TMPDIR": true, "TMP": true, "TEMP": true, "LANG": true, "LC_ALL": true, "SystemRoot": true, "ComSpec": true, "PATHEXT": true}
	result := make([]string, 0, len(allowed))
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if allowed[key] {
			result = append(result, entry)
		}
	}
	return result
}
