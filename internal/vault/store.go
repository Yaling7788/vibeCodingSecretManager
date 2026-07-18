package vault

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Yaling7788/vibeCodingSecretManager/internal/model"
	"github.com/Yaling7788/vibeCodingSecretManager/internal/securecrypto"
	_ "modernc.org/sqlite"
)

const schemaVersion = "2"

var (
	ErrLocked          = errors.New("vault is locked")
	ErrAlreadyExists   = errors.New("object already exists")
	ErrNotFound        = errors.New("object not found")
	ErrInvalidPassword = errors.New("master password is incorrect or vault wrapper is damaged")
)

type keySet struct {
	dek      []byte
	metadata []byte
	lookup   []byte
	audit    []byte
}

type passwordWrapper struct {
	Params     securecrypto.KDFParams
	Salt       []byte
	Nonce      []byte
	Ciphertext []byte
}

type Store struct {
	path string
	db   *sql.DB

	mu      sync.RWMutex
	vaultID string
	keys    *keySet
}

func Initialize(ctx context.Context, path string, password []byte, params securecrypto.KDFParams) (*Store, error) {
	if err := securecrypto.ValidateKDFParams(params); err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); err == nil {
		return nil, fmt.Errorf("vault already exists at %s", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create vault directory: %w", err)
	}

	store, err := openDatabase(path)
	if err != nil {
		return nil, err
	}
	removeOnError := true
	defer func() {
		if removeOnError {
			_ = store.Close()
			_ = os.Remove(path)
		}
	}()

	if err := store.createSchema(ctx); err != nil {
		return nil, err
	}
	vaultID, err := randomID()
	if err != nil {
		return nil, err
	}
	dek, err := securecrypto.RandomBytes(securecrypto.KeySize)
	if err != nil {
		return nil, err
	}
	defer securecrypto.Zero(dek)
	salt, err := securecrypto.RandomBytes(securecrypto.SaltSize)
	if err != nil {
		return nil, err
	}
	kek, err := securecrypto.DeriveKEK(password, salt, params)
	if err != nil {
		return nil, err
	}
	defer securecrypto.Zero(kek)
	wrapKey, err := securecrypto.ExpandKey(kek, "password-wrapper")
	if err != nil {
		return nil, err
	}
	defer securecrypto.Zero(wrapKey)
	nonce, ciphertext, err := securecrypto.Seal(wrapKey, dek, []byte(wrapperAAD(vaultID)))
	if err != nil {
		return nil, err
	}
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	for key, value := range map[string]string{
		"schema_version": schemaVersion,
		"vault_id":       vaultID,
		"created_at":     time.Now().UTC().Format(time.RFC3339Nano),
	} {
		if _, err := tx.ExecContext(ctx, `INSERT INTO vault_metadata(key, value) VALUES(?, ?)`, key, value); err != nil {
			return nil, err
		}
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO key_wrappers(wrapper_id, wrapper_type, params, salt, nonce, ciphertext, active, created_at)
		VALUES(?, 'password-argon2id', ?, ?, ?, ?, 1, ?)`,
		"password-primary", paramsJSON, salt, nonce, ciphertext, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return nil, fmt.Errorf("secure vault permissions: %w", err)
	}
	store.vaultID = vaultID
	if err := store.installKeys(dek); err != nil {
		return nil, err
	}
	removeOnError = false
	return store, nil
}

func Open(path string) (*Store, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("open vault: %w", err)
	}
	store, err := openDatabase(path)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var version string
	if err := store.db.QueryRowContext(ctx, `SELECT value FROM vault_metadata WHERE key='schema_version'`).Scan(&version); err != nil {
		store.Close()
		return nil, fmt.Errorf("read vault schema: %w", err)
	}
	if version != schemaVersion {
		store.Close()
		return nil, fmt.Errorf("unsupported vault schema %q", version)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT value FROM vault_metadata WHERE key='vault_id'`).Scan(&store.vaultID); err != nil {
		store.Close()
		return nil, fmt.Errorf("read vault id: %w", err)
	}
	return store, nil
}

func openDatabase(path string) (*Store, error) {
	dsn := "file:" + filepath.ToSlash(path) + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=secure_delete(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite vault: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize sqlite vault: %w", err)
	}
	return &Store{path: path, db: db}, nil
}

func (s *Store) createSchema(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE vault_metadata (key TEXT PRIMARY KEY, value BLOB NOT NULL)`,
		`CREATE TABLE key_wrappers (
			wrapper_id TEXT PRIMARY KEY,
			wrapper_type TEXT NOT NULL,
			params BLOB NOT NULL,
			salt BLOB NOT NULL,
			nonce BLOB NOT NULL,
			ciphertext BLOB NOT NULL,
			active INTEGER NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE encrypted_objects (
			object_id TEXT PRIMARY KEY,
			object_type TEXT NOT NULL,
			lookup_token TEXT NOT NULL,
			metadata_nonce BLOB NOT NULL,
			metadata_ciphertext BLOB NOT NULL,
			revision INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE(object_type, lookup_token)
		)`,
		`CREATE TABLE secret_versions (
			version_id TEXT PRIMARY KEY,
			secret_id TEXT NOT NULL REFERENCES encrypted_objects(object_id) ON DELETE CASCADE,
			version_number INTEGER NOT NULL,
			state TEXT NOT NULL,
			value_nonce BLOB NOT NULL,
			value_ciphertext BLOB NOT NULL,
			created_at TEXT NOT NULL,
			activated_at TEXT,
			retired_at TEXT,
			UNIQUE(secret_id, version_number)
		)`,
		`CREATE TABLE audit_events (
			event_id TEXT PRIMARY KEY,
			event_json BLOB NOT NULL,
			previous_hmac BLOB,
			event_hmac BLOB NOT NULL,
			created_at TEXT NOT NULL
		)`,
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("create vault schema: %w", err)
		}
	}
	return tx.Commit()
}

func (s *Store) Unlock(ctx context.Context, password []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.keys != nil {
		return nil
	}
	dek, err := s.unwrapDEK(ctx, password)
	if err != nil {
		return err
	}
	defer securecrypto.Zero(dek)
	return s.installKeysLocked(dek)
}

// VerifyPassword checks the password wrapper even when the vault is already
// unlocked. It is used for user-approved administrative changes.
func (s *Store) VerifyPassword(ctx context.Context, password []byte) error {
	dek, err := s.unwrapDEK(ctx, password)
	if err != nil {
		return err
	}
	securecrypto.Zero(dek)
	return nil
}

func (s *Store) unwrapDEK(ctx context.Context, password []byte) ([]byte, error) {
	var paramsJSON []byte
	var wrapper passwordWrapper
	if err := s.db.QueryRowContext(ctx, `
		SELECT params, salt, nonce, ciphertext FROM key_wrappers
		WHERE wrapper_type='password-argon2id' AND active=1
		ORDER BY created_at DESC LIMIT 1`).Scan(&paramsJSON, &wrapper.Salt, &wrapper.Nonce, &wrapper.Ciphertext); err != nil {
		return nil, fmt.Errorf("read password wrapper: %w", err)
	}
	if err := json.Unmarshal(paramsJSON, &wrapper.Params); err != nil {
		return nil, fmt.Errorf("parse password wrapper: %w", err)
	}
	kek, err := securecrypto.DeriveKEK(password, wrapper.Salt, wrapper.Params)
	if err != nil {
		return nil, err
	}
	defer securecrypto.Zero(kek)
	wrapKey, err := securecrypto.ExpandKey(kek, "password-wrapper")
	if err != nil {
		return nil, err
	}
	defer securecrypto.Zero(wrapKey)
	dek, err := securecrypto.Open(wrapKey, wrapper.Nonce, wrapper.Ciphertext, []byte(wrapperAAD(s.vaultID)))
	if err != nil {
		return nil, ErrInvalidPassword
	}
	return dek, nil
}

func (s *Store) installKeys(dek []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.installKeysLocked(dek)
}

func (s *Store) installKeysLocked(dek []byte) error {
	metadata, err := securecrypto.ExpandKey(dek, "metadata")
	if err != nil {
		return err
	}
	lookup, err := securecrypto.ExpandKey(dek, "lookup")
	if err != nil {
		securecrypto.Zero(metadata)
		return err
	}
	audit, err := securecrypto.ExpandKey(dek, "audit")
	if err != nil {
		securecrypto.Zero(metadata)
		securecrypto.Zero(lookup)
		return err
	}
	dekCopy := append([]byte(nil), dek...)
	s.keys = &keySet{dek: dekCopy, metadata: metadata, lookup: lookup, audit: audit}
	return nil
}

func (s *Store) Lock() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.keys == nil {
		return
	}
	securecrypto.Zero(s.keys.dek)
	securecrypto.Zero(s.keys.metadata)
	securecrypto.Zero(s.keys.lookup)
	securecrypto.Zero(s.keys.audit)
	s.keys = nil
}

func (s *Store) IsUnlocked() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.keys != nil
}

func (s *Store) VaultID() string { return s.vaultID }
func (s *Store) Path() string    { return s.path }

func (s *Store) Close() error {
	s.Lock()
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Store) CreateSecret(ctx context.Context, meta model.SecretMetadata, value []byte) (model.SecretMetadata, error) {
	keys, err := s.copyKeys()
	if err != nil {
		return model.SecretMetadata{}, err
	}
	defer keys.zero()
	if meta.ID == "" {
		meta.ID, err = randomID()
		if err != nil {
			return model.SecretMetadata{}, err
		}
	}
	now := time.Now().UTC()
	meta.Version = 1
	meta.State = "active"
	meta.CreatedAt = now
	meta.UpdatedAt = now
	lookup := secretLookup(keys.lookup, meta.Project, meta.Environment, meta.Name)
	metadataJSON, err := json.Marshal(meta)
	if err != nil {
		return model.SecretMetadata{}, err
	}
	metadataNonce, metadataCiphertext, err := securecrypto.Seal(keys.metadata, metadataJSON, objectAAD("secret", meta.ID, 1))
	if err != nil {
		return model.SecretMetadata{}, err
	}
	versionID, err := randomID()
	if err != nil {
		return model.SecretMetadata{}, err
	}
	valueNonce, valueCiphertext, err := securecrypto.Seal(keys.dek, value, secretAAD(meta.ID, 1))
	if err != nil {
		return model.SecretMetadata{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.SecretMetadata{}, err
	}
	defer tx.Rollback()
	stamp := now.Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO encrypted_objects(object_id, object_type, lookup_token, metadata_nonce, metadata_ciphertext, revision, created_at, updated_at)
		VALUES(?, 'secret', ?, ?, ?, 1, ?, ?)`, meta.ID, lookup, metadataNonce, metadataCiphertext, stamp, stamp); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return model.SecretMetadata{}, ErrAlreadyExists
		}
		return model.SecretMetadata{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO secret_versions(version_id, secret_id, version_number, state, value_nonce, value_ciphertext, created_at, activated_at)
		VALUES(?, ?, 1, 'active', ?, ?, ?, ?)`, versionID, meta.ID, valueNonce, valueCiphertext, stamp, stamp); err != nil {
		return model.SecretMetadata{}, err
	}
	if err := s.appendAudit(ctx, tx, keys.audit, model.AuditEvent{
		Actor: "broker", Operation: "secret.create", ObjectID: meta.ID, Outcome: "success",
	}); err != nil {
		return model.SecretMetadata{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.SecretMetadata{}, err
	}
	return meta, nil
}

func (s *Store) RotateSecret(ctx context.Context, scope model.Scope, name, profile string, strength int, value []byte) (model.SecretMetadata, error) {
	keys, err := s.copyKeys()
	if err != nil {
		return model.SecretMetadata{}, err
	}
	defer keys.zero()
	meta, revision, err := s.findSecretMetadata(ctx, keys, scope, name)
	if err != nil {
		return model.SecretMetadata{}, err
	}
	newVersion := meta.Version + 1
	now := time.Now().UTC()
	versionID, err := randomID()
	if err != nil {
		return model.SecretMetadata{}, err
	}
	valueNonce, valueCiphertext, err := securecrypto.Seal(keys.dek, value, secretAAD(meta.ID, newVersion))
	if err != nil {
		return model.SecretMetadata{}, err
	}
	meta.Profile = profile
	meta.StrengthBits = strength
	meta.Version = newVersion
	meta.State = "active"
	meta.UpdatedAt = now
	newRevision := revision + 1
	metadataJSON, err := json.Marshal(meta)
	if err != nil {
		return model.SecretMetadata{}, err
	}
	metadataNonce, metadataCiphertext, err := securecrypto.Seal(keys.metadata, metadataJSON, objectAAD("secret", meta.ID, newRevision))
	if err != nil {
		return model.SecretMetadata{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.SecretMetadata{}, err
	}
	defer tx.Rollback()
	stamp := now.Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO secret_versions(version_id, secret_id, version_number, state, value_nonce, value_ciphertext, created_at)
		VALUES(?, ?, ?, 'pending', ?, ?, ?)`, versionID, meta.ID, newVersion, valueNonce, valueCiphertext, stamp); err != nil {
		return model.SecretMetadata{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE secret_versions SET state='retired', retired_at=? WHERE secret_id=? AND state='active'`, stamp, meta.ID); err != nil {
		return model.SecretMetadata{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE secret_versions SET state='active', activated_at=? WHERE version_id=?`, stamp, versionID); err != nil {
		return model.SecretMetadata{}, err
	}
	result, err := tx.ExecContext(ctx, `
		UPDATE encrypted_objects SET metadata_nonce=?, metadata_ciphertext=?, revision=?, updated_at=?
		WHERE object_id=? AND revision=?`, metadataNonce, metadataCiphertext, newRevision, stamp, meta.ID, revision)
	if err != nil {
		return model.SecretMetadata{}, err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return model.SecretMetadata{}, fmt.Errorf("secret changed concurrently")
	}
	if err := s.appendAudit(ctx, tx, keys.audit, model.AuditEvent{Actor: "broker", Operation: "secret.rotate", ObjectID: meta.ID, Outcome: "success"}); err != nil {
		return model.SecretMetadata{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.SecretMetadata{}, err
	}
	return meta, nil
}

func (s *Store) RevokeSecret(ctx context.Context, scope model.Scope, name string) (model.SecretMetadata, error) {
	keys, err := s.copyKeys()
	if err != nil {
		return model.SecretMetadata{}, err
	}
	defer keys.zero()
	meta, revision, err := s.findSecretMetadata(ctx, keys, scope, name)
	if err != nil {
		return model.SecretMetadata{}, err
	}
	meta.State = "revoked"
	meta.UpdatedAt = time.Now().UTC()
	newRevision := revision + 1
	encoded, _ := json.Marshal(meta)
	nonce, ciphertext, err := securecrypto.Seal(keys.metadata, encoded, objectAAD("secret", meta.ID, newRevision))
	if err != nil {
		return model.SecretMetadata{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.SecretMetadata{}, err
	}
	defer tx.Rollback()
	stamp := meta.UpdatedAt.Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `UPDATE secret_versions SET state='revoked', retired_at=? WHERE secret_id=? AND state='active'`, stamp, meta.ID); err != nil {
		return model.SecretMetadata{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE encrypted_objects SET metadata_nonce=?, metadata_ciphertext=?, revision=?, updated_at=? WHERE object_id=?`, nonce, ciphertext, newRevision, stamp, meta.ID); err != nil {
		return model.SecretMetadata{}, err
	}
	if err := s.appendAudit(ctx, tx, keys.audit, model.AuditEvent{Actor: "broker", Operation: "secret.revoke", ObjectID: meta.ID, Outcome: "success"}); err != nil {
		return model.SecretMetadata{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.SecretMetadata{}, err
	}
	return meta, nil
}

func (s *Store) ListSecrets(ctx context.Context, scope model.Scope) ([]model.SecretMetadata, error) {
	keys, err := s.copyKeys()
	if err != nil {
		return nil, err
	}
	defer keys.zero()
	rows, err := s.db.QueryContext(ctx, `SELECT object_id, metadata_nonce, metadata_ciphertext, revision FROM encrypted_objects WHERE object_type='secret'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.SecretMetadata
	for rows.Next() {
		var id string
		var nonce, ciphertext []byte
		var revision int
		if err := rows.Scan(&id, &nonce, &ciphertext, &revision); err != nil {
			return nil, err
		}
		plaintext, err := securecrypto.Open(keys.metadata, nonce, ciphertext, objectAAD("secret", id, revision))
		if err != nil {
			return nil, fmt.Errorf("decrypt secret metadata %s: %w", id, err)
		}
		var meta model.SecretMetadata
		if err := json.Unmarshal(plaintext, &meta); err != nil {
			securecrypto.Zero(plaintext)
			return nil, err
		}
		securecrypto.Zero(plaintext)
		if meta.Project == scope.Project && meta.Environment == scope.Environment {
			result = append(result, meta)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, rows.Err()
}

func (s *Store) ResolveSecrets(ctx context.Context, scope model.Scope, mappings map[string]string) (map[string][]byte, error) {
	keys, err := s.copyKeys()
	if err != nil {
		return nil, err
	}
	defer keys.zero()
	result := make(map[string][]byte, len(mappings))
	for envName, secretName := range mappings {
		lookup := secretLookup(keys.lookup, scope.Project, scope.Environment, secretName)
		var id string
		if err := s.db.QueryRowContext(ctx, `SELECT object_id FROM encrypted_objects WHERE object_type='secret' AND lookup_token=?`, lookup).Scan(&id); err != nil {
			zeroMap(result)
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("secret %s: %w", secretName, ErrNotFound)
			}
			return nil, err
		}
		var version int
		var nonce, ciphertext []byte
		if err := s.db.QueryRowContext(ctx, `
			SELECT version_number, value_nonce, value_ciphertext FROM secret_versions
			WHERE secret_id=? AND state='active' ORDER BY version_number DESC LIMIT 1`, id).Scan(&version, &nonce, &ciphertext); err != nil {
			zeroMap(result)
			return nil, fmt.Errorf("active version for %s: %w", secretName, err)
		}
		plaintext, err := securecrypto.Open(keys.dek, nonce, ciphertext, secretAAD(id, version))
		if err != nil {
			zeroMap(result)
			return nil, fmt.Errorf("decrypt secret %s: %w", secretName, err)
		}
		result[envName] = plaintext
	}
	return result, nil
}

func (s *Store) PutAction(ctx context.Context, action model.Action) (model.Action, error) {
	keys, err := s.copyKeys()
	if err != nil {
		return model.Action{}, err
	}
	defer keys.zero()
	if action.Project == "" || action.Environment == "" || action.Name == "" || action.Executable == "" {
		return model.Action{}, fmt.Errorf("project, environment, name, and executable are required")
	}
	now := time.Now().UTC()
	lookup := actionLookup(keys.lookup, action.Project, action.Environment, action.Name)
	var existingID string
	var revision int
	err = s.db.QueryRowContext(ctx, `SELECT object_id, revision FROM encrypted_objects WHERE object_type='action' AND lookup_token=?`, lookup).Scan(&existingID, &revision)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return model.Action{}, err
	}
	if errors.Is(err, sql.ErrNoRows) {
		action.ID, err = randomID()
		action.CreatedAt = now
		revision = 0
	} else {
		action.ID = existingID
		var current model.Action
		current, _, err = s.getActionByID(ctx, keys, existingID, revision)
		if err != nil {
			return model.Action{}, err
		}
		action.CreatedAt = current.CreatedAt
	}
	action.UpdatedAt = now
	newRevision := revision + 1
	encoded, err := json.Marshal(action)
	if err != nil {
		return model.Action{}, err
	}
	nonce, ciphertext, err := securecrypto.Seal(keys.metadata, encoded, objectAAD("action", action.ID, newRevision))
	if err != nil {
		return model.Action{}, err
	}
	stamp := now.Format(time.RFC3339Nano)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.Action{}, err
	}
	defer tx.Rollback()
	if revision == 0 {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO encrypted_objects(object_id, object_type, lookup_token, metadata_nonce, metadata_ciphertext, revision, created_at, updated_at)
			VALUES(?, 'action', ?, ?, ?, 1, ?, ?)`, action.ID, lookup, nonce, ciphertext, stamp, stamp)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE encrypted_objects SET metadata_nonce=?, metadata_ciphertext=?, revision=?, updated_at=? WHERE object_id=?`, nonce, ciphertext, newRevision, stamp, action.ID)
	}
	if err != nil {
		return model.Action{}, err
	}
	if err := s.appendAudit(ctx, tx, keys.audit, model.AuditEvent{Actor: "user", Operation: "action.configure", ObjectID: action.ID, Outcome: "success"}); err != nil {
		return model.Action{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.Action{}, err
	}
	return action, nil
}

func (s *Store) GetAction(ctx context.Context, scope model.Scope, name string) (model.Action, error) {
	keys, err := s.copyKeys()
	if err != nil {
		return model.Action{}, err
	}
	defer keys.zero()
	lookup := actionLookup(keys.lookup, scope.Project, scope.Environment, name)
	var id string
	var revision int
	if err := s.db.QueryRowContext(ctx, `SELECT object_id, revision FROM encrypted_objects WHERE object_type='action' AND lookup_token=?`, lookup).Scan(&id, &revision); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Action{}, ErrNotFound
		}
		return model.Action{}, err
	}
	action, _, err := s.getActionByID(ctx, keys, id, revision)
	return action, err
}

func (s *Store) ListActions(ctx context.Context, scope model.Scope) ([]model.Action, error) {
	keys, err := s.copyKeys()
	if err != nil {
		return nil, err
	}
	defer keys.zero()
	rows, err := s.db.QueryContext(ctx, `SELECT object_id, metadata_nonce, metadata_ciphertext, revision FROM encrypted_objects WHERE object_type='action'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.Action
	for rows.Next() {
		var id string
		var nonce, ciphertext []byte
		var revision int
		if err := rows.Scan(&id, &nonce, &ciphertext, &revision); err != nil {
			return nil, err
		}
		plaintext, err := securecrypto.Open(keys.metadata, nonce, ciphertext, objectAAD("action", id, revision))
		if err != nil {
			return nil, err
		}
		var action model.Action
		err = json.Unmarshal(plaintext, &action)
		securecrypto.Zero(plaintext)
		if err != nil {
			return nil, err
		}
		if action.Project == scope.Project && action.Environment == scope.Environment {
			result = append(result, action)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, rows.Err()
}

func (s *Store) ListAudit(ctx context.Context, limit int) ([]model.AuditEvent, error) {
	keys, err := s.copyKeys()
	if err != nil {
		return nil, err
	}
	defer keys.zero()
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT event_json, previous_hmac, event_hmac FROM audit_events ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.AuditEvent
	var previous []byte
	for rows.Next() {
		var encoded, storedPrevious, signature []byte
		if err := rows.Scan(&encoded, &storedPrevious, &signature); err != nil {
			return nil, err
		}
		if !hmac.Equal(storedPrevious, previous) {
			return nil, fmt.Errorf("audit chain is inconsistent")
		}
		mac := hmac.New(sha256.New, keys.audit)
		_, _ = mac.Write(previous)
		_, _ = mac.Write(encoded)
		if !hmac.Equal(signature, mac.Sum(nil)) {
			return nil, fmt.Errorf("audit event authentication failed")
		}
		var event model.AuditEvent
		if err := json.Unmarshal(encoded, &event); err != nil {
			return nil, err
		}
		result = append(result, event)
		previous = append(previous[:0], signature...)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(result) > limit {
		result = result[len(result)-limit:]
	}
	for left, right := 0, len(result)-1; left < right; left, right = left+1, right-1 {
		result[left], result[right] = result[right], result[left]
	}
	return result, nil
}

func (s *Store) RecordAudit(ctx context.Context, event model.AuditEvent) error {
	keys, err := s.copyKeys()
	if err != nil {
		return err
	}
	defer keys.zero()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.appendAudit(ctx, tx, keys.audit, event); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) findSecretMetadata(ctx context.Context, keys *keySet, scope model.Scope, name string) (model.SecretMetadata, int, error) {
	lookup := secretLookup(keys.lookup, scope.Project, scope.Environment, name)
	var id string
	var nonce, ciphertext []byte
	var revision int
	if err := s.db.QueryRowContext(ctx, `
		SELECT object_id, metadata_nonce, metadata_ciphertext, revision FROM encrypted_objects
		WHERE object_type='secret' AND lookup_token=?`, lookup).Scan(&id, &nonce, &ciphertext, &revision); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.SecretMetadata{}, 0, ErrNotFound
		}
		return model.SecretMetadata{}, 0, err
	}
	plaintext, err := securecrypto.Open(keys.metadata, nonce, ciphertext, objectAAD("secret", id, revision))
	if err != nil {
		return model.SecretMetadata{}, 0, err
	}
	defer securecrypto.Zero(plaintext)
	var meta model.SecretMetadata
	if err := json.Unmarshal(plaintext, &meta); err != nil {
		return model.SecretMetadata{}, 0, err
	}
	return meta, revision, nil
}

func (s *Store) getActionByID(ctx context.Context, keys *keySet, id string, revision int) (model.Action, int, error) {
	var nonce, ciphertext []byte
	if err := s.db.QueryRowContext(ctx, `SELECT metadata_nonce, metadata_ciphertext FROM encrypted_objects WHERE object_id=?`, id).Scan(&nonce, &ciphertext); err != nil {
		return model.Action{}, 0, err
	}
	plaintext, err := securecrypto.Open(keys.metadata, nonce, ciphertext, objectAAD("action", id, revision))
	if err != nil {
		return model.Action{}, 0, err
	}
	defer securecrypto.Zero(plaintext)
	var action model.Action
	if err := json.Unmarshal(plaintext, &action); err != nil {
		return model.Action{}, 0, err
	}
	return action, revision, nil
}

func (s *Store) appendAudit(ctx context.Context, tx *sql.Tx, key []byte, event model.AuditEvent) error {
	if event.ID == "" {
		var err error
		event.ID, err = randomID()
		if err != nil {
			return err
		}
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		return err
	}
	var previous []byte
	err = tx.QueryRowContext(ctx, `SELECT event_hmac FROM audit_events ORDER BY created_at DESC LIMIT 1`).Scan(&previous)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(previous)
	_, _ = mac.Write(encoded)
	signature := mac.Sum(nil)
	_, err = tx.ExecContext(ctx, `INSERT INTO audit_events(event_id, event_json, previous_hmac, event_hmac, created_at) VALUES(?, ?, ?, ?, ?)`, event.ID, encoded, previous, signature, event.Timestamp.Format(time.RFC3339Nano))
	return err
}

func (s *Store) copyKeys() (*keySet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.keys == nil {
		return nil, ErrLocked
	}
	return &keySet{
		dek:      append([]byte(nil), s.keys.dek...),
		metadata: append([]byte(nil), s.keys.metadata...),
		lookup:   append([]byte(nil), s.keys.lookup...),
		audit:    append([]byte(nil), s.keys.audit...),
	}, nil
}

func (k *keySet) zero() {
	securecrypto.Zero(k.dek)
	securecrypto.Zero(k.metadata)
	securecrypto.Zero(k.lookup)
	securecrypto.Zero(k.audit)
}

func wrapperAAD(vaultID string) string { return "vcsm:dek:v1:" + vaultID }
func objectAAD(kind, id string, revision int) []byte {
	return []byte(fmt.Sprintf("vcsm:object:v1:%s:%s:%d", kind, id, revision))
}
func secretAAD(id string, version int) []byte {
	return []byte(fmt.Sprintf("vcsm:secret:v1:%s:%d", id, version))
}
func secretLookup(key []byte, project, environment, name string) string {
	return securecrypto.LookupToken(key, "secret", project, environment, name)
}
func actionLookup(key []byte, project, environment, name string) string {
	return securecrypto.LookupToken(key, "action", project, environment, name)
}

func randomID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	hexValue := hex.EncodeToString(value)
	return fmt.Sprintf("%s-%s-%s-%s-%s", hexValue[:8], hexValue[8:12], hexValue[12:16], hexValue[16:20], hexValue[20:]), nil
}

func zeroMap(values map[string][]byte) {
	for key, value := range values {
		securecrypto.Zero(value)
		delete(values, key)
	}
}
