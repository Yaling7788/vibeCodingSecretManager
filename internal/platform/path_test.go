package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPathHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	got, err := ExpandPath("~/Secrets/file.kdbx")
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(home, "Secrets", "file.kdbx")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
