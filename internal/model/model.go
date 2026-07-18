package model

import "time"

type Scope struct {
	Project     string `json:"project" yaml:"project"`
	Environment string `json:"environment" yaml:"environment"`
}

func (s Scope) Key() string {
	return s.Project + "\x00" + s.Environment
}

type SecretMetadata struct {
	ID           string    `json:"id"`
	Project      string    `json:"project"`
	Environment  string    `json:"environment"`
	Name         string    `json:"name"`
	Profile      string    `json:"profile"`
	StrengthBits int       `json:"strength_bits"`
	Version      int       `json:"version"`
	State        string    `json:"state"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type SecretStatus struct {
	SecretMetadata
	Available bool `json:"available"`
}

type Action struct {
	ID           string            `json:"id" yaml:"id"`
	Project      string            `json:"project" yaml:"project"`
	Environment  string            `json:"environment" yaml:"environment"`
	Name         string            `json:"name" yaml:"name"`
	Executable   string            `json:"executable" yaml:"executable"`
	Arguments    []string          `json:"arguments" yaml:"arguments"`
	Directory    string            `json:"directory" yaml:"directory"`
	Secrets      map[string]string `json:"secrets" yaml:"secrets"`
	OutputPolicy string            `json:"output_policy" yaml:"output_policy"`
	CreatedAt    time.Time         `json:"created_at" yaml:"-"`
	UpdatedAt    time.Time         `json:"updated_at" yaml:"-"`
}

type AuditEvent struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Actor     string    `json:"actor"`
	Operation string    `json:"operation"`
	ObjectID  string    `json:"object_id,omitempty"`
	Outcome   string    `json:"outcome"`
	Detail    string    `json:"detail,omitempty"`
}
