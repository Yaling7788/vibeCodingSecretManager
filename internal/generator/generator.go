package generator

import (
	"crypto/rand"
	"fmt"
	"math"
)

type Profile struct {
	Name         string
	Alphabet     string
	Length       int
	StrengthBits int
}

var profiles = map[string]Profile{
	// URL-safe base64 alphabet: broadly accepted in headers, URLs, env vars and
	// configuration formats while still providing at least 256 bits of entropy.
	"default": {Name: "default", Alphabet: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_", Length: 43, StrengthBits: 258},
	"alnum":   {Name: "alnum", Alphabet: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789", Length: 44, StrengthBits: 262},
	"password": {Name: "password", Alphabet: "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789-_.", Length: 40,
		StrengthBits: 238},
	"database": {Name: "database", Alphabet: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789", Length: 44, StrengthBits: 262},
}

func Get(name string) (Profile, error) {
	if name == "" {
		name = "default"
	}
	profile, ok := profiles[name]
	if !ok {
		return Profile{}, fmt.Errorf("unknown generation profile %q", name)
	}
	return profile, nil
}

func Generate(name string) ([]byte, Profile, error) {
	profile, err := Get(name)
	if err != nil {
		return nil, Profile{}, err
	}
	value := make([]byte, profile.Length)
	// Rejection sampling avoids modulo bias.
	limit := 256 - (256 % len(profile.Alphabet))
	random := make([]byte, profile.Length*2)
	for written := 0; written < len(value); {
		if _, err := rand.Read(random); err != nil {
			return nil, Profile{}, err
		}
		for _, candidate := range random {
			if int(candidate) >= limit {
				continue
			}
			value[written] = profile.Alphabet[int(candidate)%len(profile.Alphabet)]
			written++
			if written == len(value) {
				break
			}
		}
	}
	profile.StrengthBits = int(math.Floor(float64(profile.Length) * math.Log2(float64(len(profile.Alphabet)))))
	return value, profile, nil
}
