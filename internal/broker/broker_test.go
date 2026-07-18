//go:build !windows

package broker

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Yaling7788/vibeCodingSecretManager/internal/api"
	"github.com/Yaling7788/vibeCodingSecretManager/internal/model"
	"github.com/Yaling7788/vibeCodingSecretManager/internal/securecrypto"
	"github.com/Yaling7788/vibeCodingSecretManager/internal/vault"
)

func TestProtectedWorkflowReturnsMetadataAndRedactsProcessOutput(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "vault.db")
	password := []byte("correct horse battery staple")
	store, err := vault.Initialize(ctx, path, password, securecrypto.DefaultKDFParams())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	b, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	invoke := func(operation string, payload any) []api.Response {
		encoded, _ := json.Marshal(payload)
		var responses []api.Response
		b.Handle(ctx, api.Request{ID: "test", Operation: operation, Payload: encoded}, func(response api.Response) error {
			responses = append(responses, response)
			return nil
		})
		if len(responses) == 0 || !responses[len(responses)-1].OK {
			t.Fatalf("%s failed: %+v", operation, responses)
		}
		return responses
	}

	invoke("unlock", api.UnlockRequest{Password: append([]byte(nil), password...)})
	scope := model.Scope{Project: "sample", Environment: "dev"}
	created := invoke("secret.create", api.SecretRequest{Scope: scope, Name: "api-token", Profile: "default"})
	if strings.Contains(string(created[len(created)-1].Data), "correct horse") {
		t.Fatal("response contained authentication material")
	}
	action := model.Action{Project: "sample", Environment: "dev", Name: "redaction-test", Executable: "/bin/sh", Arguments: []string{"-c", `printf '%s' "$TOKEN"`}, Secrets: map[string]string{"TOKEN": "api-token"}, OutputPolicy: "redacted"}
	invoke("action.configure", api.ConfigureActionRequest{Action: action, MasterPassword: append([]byte(nil), password...)})
	responses := invoke("run", api.ActionRequest{Scope: scope, Name: action.Name})
	var streamed strings.Builder
	for _, response := range responses {
		if response.Event != "stdout" {
			continue
		}
		var text string
		if err := json.Unmarshal(response.Data, &text); err != nil {
			t.Fatal(err)
		}
		streamed.WriteString(text)
	}
	if streamed.String() != "[REDACTED]" {
		t.Fatalf("unexpected redacted output %q", streamed.String())
	}
}
