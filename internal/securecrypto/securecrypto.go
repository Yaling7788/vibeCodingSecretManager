package securecrypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

const (
	KeySize   = 32
	SaltSize  = 16
	NonceSize = chacha20poly1305.NonceSizeX
)

type KDFParams struct {
	MemoryKiB   uint32 `json:"memory_kib"`
	Iterations  uint32 `json:"iterations"`
	Parallelism uint8  `json:"parallelism"`
}

func DefaultKDFParams() KDFParams {
	return KDFParams{MemoryKiB: 64 * 1024, Iterations: 3, Parallelism: 1}
}

func ValidateKDFParams(p KDFParams) error {
	if p.MemoryKiB < 19*1024 {
		return fmt.Errorf("argon2id memory must be at least 19 MiB")
	}
	if p.Iterations < 2 {
		return fmt.Errorf("argon2id iterations must be at least 2")
	}
	if p.Parallelism == 0 {
		return fmt.Errorf("argon2id parallelism must be positive")
	}
	return nil
}

func RandomBytes(size int) ([]byte, error) {
	value := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, value); err != nil {
		return nil, fmt.Errorf("generate random bytes: %w", err)
	}
	return value, nil
}

func DeriveKEK(password, salt []byte, params KDFParams) ([]byte, error) {
	if len(password) == 0 {
		return nil, fmt.Errorf("master password is empty")
	}
	if len(password) > 4096 {
		return nil, fmt.Errorf("master password exceeds 4096 bytes")
	}
	if len(salt) < SaltSize {
		return nil, fmt.Errorf("argon2id salt is too short")
	}
	if err := ValidateKDFParams(params); err != nil {
		return nil, err
	}
	return argon2.IDKey(password, salt, params.Iterations, params.MemoryKiB, params.Parallelism, KeySize), nil
}

func ExpandKey(key []byte, purpose string) ([]byte, error) {
	reader := hkdf.New(sha256.New, key, nil, []byte("vcsm:v2:"+purpose))
	out := make([]byte, KeySize)
	if _, err := io.ReadFull(reader, out); err != nil {
		return nil, fmt.Errorf("derive %s key: %w", purpose, err)
	}
	return out, nil
}

func Seal(key, plaintext, associatedData []byte) (nonce, ciphertext []byte, err error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, nil, fmt.Errorf("initialize encryption: %w", err)
	}
	nonce, err = RandomBytes(aead.NonceSize())
	if err != nil {
		return nil, nil, err
	}
	ciphertext = aead.Seal(nil, nonce, plaintext, associatedData)
	return nonce, ciphertext, nil
}

func Open(key, nonce, ciphertext, associatedData []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("initialize decryption: %w", err)
	}
	if len(nonce) != aead.NonceSize() {
		return nil, fmt.Errorf("invalid nonce length")
	}
	plaintext, err := aead.Open(nil, nonce, ciphertext, associatedData)
	if err != nil {
		return nil, fmt.Errorf("authentication failed")
	}
	return plaintext, nil
}

func LookupToken(key []byte, values ...string) string {
	mac := hmac.New(sha256.New, key)
	for _, value := range values {
		_, _ = mac.Write([]byte{0})
		_, _ = mac.Write([]byte(value))
	}
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func Zero(value []byte) {
	for i := range value {
		value[i] = 0
	}
}
