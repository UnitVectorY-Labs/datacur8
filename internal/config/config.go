package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version    string      `yaml:"version"`
	StrictMode string      `yaml:"strict_mode,omitempty"`
	Types      []TypeDef   `yaml:"types"`
	Tidy       *TidyConfig `yaml:"tidy,omitempty"`
}

type TypeDef struct {
	Name        string          `yaml:"name"`
	Input       string          `yaml:"input"`
	Match       MatchDef        `yaml:"match"`
	Schema      map[string]any  `yaml:"schema"`
	Constraints []ConstraintDef `yaml:"constraints,omitempty"`
	Output      *OutputDef      `yaml:"output,omitempty"`
	CSV         *CSVDef         `yaml:"csv,omitempty"`
	Tidy        *TypeTidyDef    `yaml:"tidy,omitempty"`
}

type MatchDef struct {
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude,omitempty"`
}

type CSVDef struct {
	Delimiter string `yaml:"delimiter,omitempty"`
}

type TypeTidyDef struct {
	SortArraysBy []string `yaml:"sort_arrays_by,omitempty"`
}

type OutputDef struct {
	Path   string `yaml:"path"`
	Format string `yaml:"format"`
}

type ConstraintDef struct {
	ID            string        `yaml:"id,omitempty"`
	Type          string        `yaml:"type"`
	Key           string        `yaml:"key,omitempty"`
	CaseSensitive *bool         `yaml:"case_sensitive,omitempty"`
	Scope         string        `yaml:"scope,omitempty"`
	PathSelector  string        `yaml:"path_selector,omitempty"`
	References    *ReferenceDef `yaml:"references,omitempty"`
}

type ReferenceDef struct {
	Type string `yaml:"type,omitempty"`
	Key  string `yaml:"key,omitempty"`
}

type TidyConfig struct {
	Enabled *bool `yaml:"enabled,omitempty"`
}

// Load reads and parses a .datacur8 YAML config file at the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	if err := validateAgainstEmbeddedSchema(data); err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	cfg.Defaults()
	return &cfg, nil
}

// Defaults applies default values to the config where fields are unset.
func (c *Config) Defaults() {
	if c.StrictMode == "" {
		c.StrictMode = "DISABLED"
	}

	for i := range c.Types {
		t := &c.Types[i]

		if t.CSV != nil && t.CSV.Delimiter == "" {
			t.CSV.Delimiter = ","
		}

		for j := range t.Constraints {
			con := &t.Constraints[j]
			if con.Scope == "" {
				con.Scope = "type"
			}
		}
	}
}

// IsCaseSensitive returns true if case_sensitive is nil (unset) or explicitly true.
func (c *ConstraintDef) IsCaseSensitive() bool {
	return c.CaseSensitive == nil || *c.CaseSensitive
}

// IsEnabled returns true if the TidyConfig is nil, Enabled is nil (unset), or explicitly true.
func (t *TidyConfig) IsEnabled() bool {
	return t == nil || t.Enabled == nil || *t.Enabled
}
