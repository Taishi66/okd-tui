package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var DefaultProdPatterns = []string{"prod", "production", "prd", "live"}

// AppConfig holds all configuration for okd-tui.
type AppConfig struct {
	ProdPatterns       []string    `yaml:"prod_patterns"`
	ReadonlyNamespaces []string    `yaml:"readonly_namespaces"`
	Cache              CacheConfig `yaml:"cache"`
	Exec               ExecConfig  `yaml:"exec"`
}

// CacheConfig holds TTL settings for cached resources.
type CacheConfig struct {
	PodsTTL        time.Duration `yaml:"pods"`
	NamespacesTTL  time.Duration `yaml:"namespaces"`
	DeploymentsTTL time.Duration `yaml:"deployments"`
	EventsTTL      time.Duration `yaml:"events"`
}

// ExecConfig holds exec/shell settings.
type ExecConfig struct {
	Shell string `yaml:"shell"`
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() *AppConfig {
	return &AppConfig{
		ProdPatterns:       DefaultProdPatterns,
		ReadonlyNamespaces: nil,
		Cache: CacheConfig{
			PodsTTL:        5 * time.Second,
			NamespacesTTL:  30 * time.Second,
			DeploymentsTTL: 10 * time.Second,
			EventsTTL:      10 * time.Second,
		},
		Exec: ExecConfig{
			Shell: "/bin/sh",
		},
	}
}

// LoadConfig loads from the default path ~/.config/okd-tui/config.yaml.
func LoadConfig() (*AppConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return DefaultConfig(), nil
	}
	return LoadConfigFrom(filepath.Join(home, ".config", "okd-tui", "config.yaml"))
}

// LoadConfigFrom loads config from a specific file path.
// Returns defaults if the file does not exist.
func LoadConfigFrom(path string) (*AppConfig, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, nil
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Apply defaults for zero values
	if len(cfg.ProdPatterns) == 0 {
		cfg.ProdPatterns = DefaultProdPatterns
	}
	if cfg.Cache.PodsTTL == 0 {
		cfg.Cache.PodsTTL = 5 * time.Second
	}
	if cfg.Cache.NamespacesTTL == 0 {
		cfg.Cache.NamespacesTTL = 30 * time.Second
	}
	if cfg.Cache.DeploymentsTTL == 0 {
		cfg.Cache.DeploymentsTTL = 10 * time.Second
	}
	if cfg.Cache.EventsTTL == 0 {
		cfg.Cache.EventsTTL = 10 * time.Second
	}
	if cfg.Exec.Shell == "" {
		cfg.Exec.Shell = "/bin/sh"
	}

	return cfg, nil
}

// IsReadonlyNamespace checks if a namespace matches any readonly pattern.
// Supports glob matching (e.g. "openshift-*").
func IsReadonlyNamespace(namespace string, patterns []string) bool {
	if namespace == "" || len(patterns) == 0 {
		return false
	}
	for _, p := range patterns {
		matched, err := filepath.Match(p, namespace)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// IsProdNamespace checks if a namespace name matches production patterns.
// Matching is done by segment (split on -._) to avoid false positives
// like "product-api" matching "prod".
func IsProdNamespace(namespace string, patterns []string) bool {
	if len(patterns) == 0 {
		patterns = DefaultProdPatterns
	}
	ns := strings.ToLower(namespace)
	segments := splitSegments(ns)

	for _, p := range patterns {
		p = strings.ToLower(p)
		// Check if any segment matches the pattern exactly
		for _, seg := range segments {
			if seg == p {
				return true
			}
		}
	}
	return false
}

// splitSegments splits a namespace name on common separators.
func splitSegments(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '.' || r == '_'
	})
}
