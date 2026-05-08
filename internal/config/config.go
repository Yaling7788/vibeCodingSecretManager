package config

import (
	"fmt"
	"os"

	"github.com/Yaling7788/vibeCodingSecretManager/internal/platform"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Vault    Vault              `yaml:"vault"`
	Projects map[string]Project `yaml:"projects"`
}

type Vault struct {
	Type     string `yaml:"type"`
	Database string `yaml:"database"`
	KeyFile  string `yaml:"key_file"`
	CLIPath  string `yaml:"cli_path"`
}

type Project struct {
	Root         string                 `yaml:"root"`
	Environments map[string]Environment `yaml:"environments"`
}

type Environment struct {
	Secrets map[string]string `yaml:"secrets"`
}

type Selection struct {
	Project     Project
	Environment Environment
}

func Load(path string) (*Config, error) {
	expanded, err := platform.ExpandPath(path)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(expanded)
	if err != nil {
		return nil, fmt.Errorf("failed to read config %s: %w", expanded, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config %s: %w", expanded, err)
	}

	if err := cfg.Normalize(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) Normalize() error {
	if c.Vault.Type == "" {
		c.Vault.Type = "keepassxc"
	}
	if c.Vault.Type != "keepassxc" {
		return fmt.Errorf("unsupported vault type %q", c.Vault.Type)
	}

	var err error
	c.Vault.Database, err = platform.ExpandPath(c.Vault.Database)
	if err != nil {
		return err
	}
	if c.Vault.KeyFile != "" {
		c.Vault.KeyFile, err = platform.ExpandPath(c.Vault.KeyFile)
		if err != nil {
			return err
		}
	}

	for name, project := range c.Projects {
		project.Root, err = platform.ExpandPath(project.Root)
		if err != nil {
			return err
		}
		c.Projects[name] = project
	}

	return c.Validate()
}

func (c Config) Validate() error {
	if c.Vault.Database == "" {
		return fmt.Errorf("vault.database is required")
	}
	if len(c.Projects) == 0 {
		return fmt.Errorf("at least one project is required")
	}
	for projectName, project := range c.Projects {
		if project.Root == "" {
			return fmt.Errorf("project %q root is required", projectName)
		}
		if len(project.Environments) == 0 {
			return fmt.Errorf("project %q must define at least one environment", projectName)
		}
		for envName, environment := range project.Environments {
			if len(environment.Secrets) == 0 {
				return fmt.Errorf("project %q environment %q must define at least one secret", projectName, envName)
			}
			for envVar, entry := range environment.Secrets {
				if envVar == "" || entry == "" {
					return fmt.Errorf("project %q environment %q contains an empty secret mapping", projectName, envName)
				}
			}
		}
	}
	return nil
}

func (c Config) Select(projectName, environmentName string) (Selection, error) {
	project, ok := c.Projects[projectName]
	if !ok {
		return Selection{}, fmt.Errorf("project %q is not configured", projectName)
	}

	environment, ok := project.Environments[environmentName]
	if !ok {
		return Selection{}, fmt.Errorf("environment %q is not configured for project %q", environmentName, projectName)
	}

	return Selection{Project: project, Environment: environment}, nil
}

const StarterConfig = `vault:
  type: keepassxc
  database: ~/KeePass/example-dev.kdbx
  key_file: ~/KeePass/example-dev.key
  cli_path: auto

projects:
  sample-webapp:
    root: ~/Projects/sample-webapp
    environments:
      dev:
        secrets:
          DATABASE_URL: SampleWebApp/Dev/DATABASE_URL
          RESEND_API_KEY: SampleWebApp/Dev/RESEND_API_KEY
          GOOGLE_CLIENT_ID: SampleWebApp/Dev/GOOGLE_CLIENT_ID
          GOOGLE_CLIENT_SECRET: SampleWebApp/Dev/GOOGLE_CLIENT_SECRET
          ANTHROPIC_API_KEY: SampleWebApp/Dev/ANTHROPIC_API_KEY
          OPENAI_API_KEY: SampleWebApp/Dev/OPENAI_API_KEY
`
