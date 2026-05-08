package config

import "testing"

func TestSelect(t *testing.T) {
	cfg := Config{
		Vault: Vault{Type: "keepassxc", Database: "/tmp/test.kdbx"},
		Projects: map[string]Project{
			"app": {
				Root: "/tmp/app",
				Environments: map[string]Environment{
					"dev": {
						Secrets: map[string]string{"DATABASE_URL": "App/Dev/DATABASE_URL"},
					},
				},
			},
		},
	}

	if err := cfg.Normalize(); err != nil {
		t.Fatal(err)
	}

	selection, err := cfg.Select("app", "dev")
	if err != nil {
		t.Fatal(err)
	}
	if selection.Environment.Secrets["DATABASE_URL"] != "App/Dev/DATABASE_URL" {
		t.Fatalf("unexpected selection: %#v", selection)
	}
}
