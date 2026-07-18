package securecrypto

import (
	"bytes"
	"testing"
)

func testParams() KDFParams {
	return KDFParams{MemoryKiB: 19 * 1024, Iterations: 2, Parallelism: 1}
}

func TestDeriveSealOpen(t *testing.T) {
	salt := bytes.Repeat([]byte{1}, SaltSize)
	key, err := DeriveKEK([]byte("correct horse battery staple"), salt, testParams())
	if err != nil {
		t.Fatal(err)
	}
	defer Zero(key)

	nonce, ciphertext, err := Seal(key, []byte("secret"), []byte("context"))
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := Open(key, nonce, ciphertext, []byte("context"))
	if err != nil {
		t.Fatal(err)
	}
	if string(plaintext) != "secret" {
		t.Fatalf("unexpected plaintext %q", plaintext)
	}
}

func TestOpenRejectsTampering(t *testing.T) {
	key := bytes.Repeat([]byte{2}, KeySize)
	nonce, ciphertext, err := Seal(key, []byte("secret"), []byte("context"))
	if err != nil {
		t.Fatal(err)
	}
	ciphertext[0] ^= 1
	if _, err := Open(key, nonce, ciphertext, []byte("context")); err == nil {
		t.Fatal("tampered ciphertext was accepted")
	}
}
