package api

import (
	"encoding/json"
	"time"

	"github.com/Yaling7788/vibeCodingSecretManager/internal/model"
)

type Request struct {
	ID        string          `json:"id"`
	Operation string          `json:"operation"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type Response struct {
	ID    string          `json:"id"`
	OK    bool            `json:"ok"`
	Event string          `json:"event,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
	Final bool            `json:"final,omitempty"`
}

type Status struct {
	Initialized bool      `json:"initialized"`
	Unlocked    bool      `json:"unlocked"`
	VaultID     string    `json:"vault_id,omitempty"`
	StartedAt   time.Time `json:"started_at"`
	UnlockMode  string    `json:"unlock_mode"`
	PowerHold   bool      `json:"power_hold"`
}

type UnlockRequest struct {
	// Password is accepted only on the local, permission-restricted IPC endpoint
	// and is never logged or included in a response.
	Password []byte `json:"password"`
}

type SecretRequest struct {
	Scope   model.Scope `json:"scope"`
	Name    string      `json:"name"`
	Profile string      `json:"profile,omitempty"`
}

type ScopeRequest struct {
	Scope model.Scope `json:"scope"`
}

type ActionRequest struct {
	Scope model.Scope `json:"scope"`
	Name  string      `json:"name"`
}

type ConfigureActionRequest struct {
	Action         model.Action `json:"action"`
	MasterPassword []byte       `json:"master_password"`
}

type AuditRequest struct {
	Limit int `json:"limit"`
}

type RunResult struct {
	Action   string `json:"action"`
	ExitCode int    `json:"exit_code"`
}
