package generator

import (
	"strings"
	"testing"
)

func TestProfilesMeetStrengthAndCompatibility(t *testing.T) {
	for _, name := range []string{"default", "alnum", "password", "database"} {
		value, profile, err := Generate(name)
		if err != nil {
			t.Fatal(err)
		}
		if profile.StrengthBits < 200 {
			t.Fatalf("%s is too weak: %d", name, profile.StrengthBits)
		}
		if len(value) != profile.Length {
			t.Fatalf("%s length mismatch", name)
		}
		for _, char := range string(value) {
			if !strings.ContainsRune(profile.Alphabet, char) {
				t.Fatalf("%s emitted invalid character %q", name, char)
			}
		}
	}
}
