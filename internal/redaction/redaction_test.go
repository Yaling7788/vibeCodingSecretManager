package redaction

import "testing"

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
