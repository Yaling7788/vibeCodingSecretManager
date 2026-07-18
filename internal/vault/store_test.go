package vault

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/Yaling7788/vibeCodingSecretManager/internal/model"
	"github.com/Yaling7788/vibeCodingSecretManager/internal/securecrypto"
)

func testKDF() securecrypto.KDFParams {
	return securecrypto.KDFParams{MemoryKiB: 19 * 1024, Iterations: 2, Parallelism: 1}
}

func TestInitializeUnlockAndSecrets(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "vault.db")
	password := []byte("a long local master password")
	store, err := Initialize(ctx, path, password, testKDF())
	if err != nil {
		t.Fatal(err)
	}
	meta, err := store.CreateSecret(ctx, model.SecretMetadata{Project: "app", Environment: "dev", Name: "TOKEN", Profile: "token-base64url", StrengthBits: 256}, []byte("top-secret-value"))
	if err != nil {
		t.Fatal(err)
	}
	if meta.Version != 1 {
		t.Fatalf("unexpected version %d", meta.Version)
	}
	store.Close()

	store, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Unlock(ctx, []byte("wrong password")); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("wrong password error = %v", err)
	}
	if err := store.Unlock(ctx, password); err != nil {
		t.Fatal(err)
	}
	values, err := store.ResolveSecrets(ctx, model.Scope{Project: "app", Environment: "dev"}, map[string]string{"TOKEN": "TOKEN"})
	if err != nil {
		t.Fatal(err)
	}
	if string(values["TOKEN"]) != "top-secret-value" {
		t.Fatalf("unexpected value")
	}
}

func TestRotateAndRevoke(t *testing.T) {
	ctx := context.Background()
	store, err := Initialize(ctx, filepath.Join(t.TempDir(), "vault.db"), []byte("master password for tests"), testKDF())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	scope := model.Scope{Project: "app", Environment: "dev"}
	if _, err := store.CreateSecret(ctx, model.SecretMetadata{Project: "app", Environment: "dev", Name: "TOKEN", Profile: "hex", StrengthBits: 128}, []byte("old")); err != nil {
		t.Fatal(err)
	}
	meta, err := store.RotateSecret(ctx, scope, "TOKEN", "hex", 256, []byte("new"))
	if err != nil {
		t.Fatal(err)
	}
	if meta.Version != 2 {
		t.Fatalf("version=%d", meta.Version)
	}
	values, err := store.ResolveSecrets(ctx, scope, map[string]string{"TOKEN": "TOKEN"})
	if err != nil {
		t.Fatal(err)
	}
	if string(values["TOKEN"]) != "new" {
		t.Fatal("rotation did not promote new value")
	}
	meta, err = store.RevokeSecret(ctx, scope, "TOKEN")
	if err != nil {
		t.Fatal(err)
	}
	if meta.State != "revoked" {
		t.Fatalf("state=%s", meta.State)
	}
}

func TestAuditDetectsTampering(t *testing.T) {
	ctx := context.Background()
	store, err := Initialize(ctx, filepath.Join(t.TempDir(), "vault.db"), []byte("master password for tests"), testKDF())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.CreateSecret(ctx, model.SecretMetadata{Project: "app", Environment: "dev", Name: "TOKEN", Profile: "default", StrengthBits: 256}, []byte("value")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ListAudit(ctx, 10); err != nil {
		t.Fatalf("valid audit chain failed: %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE audit_events SET event_json='{}' WHERE event_id=(SELECT event_id FROM audit_events LIMIT 1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ListAudit(ctx, 10); err == nil {
		t.Fatal("tampered audit event was accepted")
	}
}
