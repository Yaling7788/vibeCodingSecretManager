package redaction

import (
	"bytes"
	"testing"
)

func TestRedact(t *testing.T) {
	got := Redact("token is abc123xyz", []string{"abc123xyz"})
	if got != "token is [REDACTED]" {
		t.Fatalf("unexpected redaction: %q", got)
	}
}

func TestRedactIgnoresTinyValues(t *testing.T) {
	got := Redact("a=b", []string{"b"})
	if got != "a=b" {
		t.Fatalf("tiny values should not be redacted: %q", got)
	}
}

func TestWriterRedactsAcrossChunks(t *testing.T) {
	var output bytes.Buffer
	w := NewWriter(&output, [][]byte{[]byte("split-secret")})
	if _, err := w.Write([]byte("value=split-")); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("secret done")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "value=[REDACTED] done" {
		t.Fatalf("unexpected output %q", got)
	}
}
