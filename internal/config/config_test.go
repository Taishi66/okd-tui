package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsProdNamespace(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		patterns  []string
		want      bool
	}{
		// Positive cases - segment matches
		{"exact prod", "prod", nil, true},
		{"segment prod", "my-app-prod", nil, true},
		{"exact production", "production", nil, true},
		{"segment production", "my-production-ns", nil, true},
		{"segment prd", "app-prd-01", nil, true},
		{"segment live", "live-env", nil, true},
		{"dot separator", "app.prod.ns", nil, true},
		{"underscore separator", "app_prod_ns", nil, true},

		// Case insensitive
		{"uppercase PROD", "MY-PROD-NS", nil, true},
		{"mixed case", "My-Prod-Namespace", nil, true},
		{"uppercase PRODUCTION", "PRODUCTION", nil, true},

		// Negative cases - no false positives
		{"dev namespace", "development", nil, false},
		{"staging", "staging", nil, false},
		{"test", "test-env", nil, false},
		{"empty namespace", "", nil, false},

		// Fixed: these were false positives before
		{"product-api NOT prod", "product-api", nil, false},
		{"reproduce NOT prod", "reproduce-bug", nil, false},
		{"productivity NOT prod", "productivity-tool", nil, false},
		{"livechat NOT live", "livechat-service", nil, false},

		// Custom patterns
		{"custom pattern match", "my-staging", []string{"staging"}, true},
		{"custom pattern no match", "my-dev", []string{"staging"}, false},
		{"empty custom patterns uses defaults", "production", []string{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsProdNamespace(tt.namespace, tt.patterns)
			if got != tt.want {
				t.Errorf("IsProdNamespace(%q, %v) = %v, want %v", tt.namespace, tt.patterns, got, tt.want)
			}
		})
	}
}

func TestIsProdNamespaceEmptyPatternsUsesDefaults(t *testing.T) {
	if !IsProdNamespace("production", []string{}) {
		t.Error("empty patterns should use defaults, but didn't match 'production'")
	}
}

// --- F-27: LoadConfig tests ---

func TestLoadConfig_Defaults(t *testing.T) {
	// Use a non-existent path so no file is loaded.
	cfg, err := LoadConfigFrom("/tmp/non-existent-okd-tui-test/config.yaml")
	if err != nil {
		t.Fatalf("LoadConfigFrom returned error: %v", err)
	}

	// ProdPatterns defaults
	if len(cfg.ProdPatterns) != len(DefaultProdPatterns) {
		t.Errorf("ProdPatterns len = %d, want %d", len(cfg.ProdPatterns), len(DefaultProdPatterns))
	}
	for i, p := range DefaultProdPatterns {
		if cfg.ProdPatterns[i] != p {
			t.Errorf("ProdPatterns[%d] = %q, want %q", i, cfg.ProdPatterns[i], p)
		}
	}

	// ReadonlyNamespaces default empty
	if len(cfg.ReadonlyNamespaces) != 0 {
		t.Errorf("ReadonlyNamespaces should be empty, got %v", cfg.ReadonlyNamespaces)
	}

	// Cache TTL defaults
	if cfg.Cache.PodsTTL != 5*time.Second {
		t.Errorf("Cache.PodsTTL = %v, want 5s", cfg.Cache.PodsTTL)
	}
	if cfg.Cache.NamespacesTTL != 30*time.Second {
		t.Errorf("Cache.NamespacesTTL = %v, want 30s", cfg.Cache.NamespacesTTL)
	}
	if cfg.Cache.DeploymentsTTL != 10*time.Second {
		t.Errorf("Cache.DeploymentsTTL = %v, want 10s", cfg.Cache.DeploymentsTTL)
	}
	if cfg.Cache.EventsTTL != 10*time.Second {
		t.Errorf("Cache.EventsTTL = %v, want 10s", cfg.Cache.EventsTTL)
	}

	// Exec defaults
	if cfg.Exec.Shell != "/bin/sh" {
		t.Errorf("Exec.Shell = %q, want /bin/sh", cfg.Exec.Shell)
	}
}

func TestLoadConfig_CustomFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	content := `prod_patterns:
  - staging
  - preprod
readonly_namespaces:
  - kube-system
  - openshift-*
cache:
  pods: 10s
  namespaces: 60s
  deployments: 20s
  events: 15s
exec:
  shell: /bin/bash
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfigFrom(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfigFrom returned error: %v", err)
	}

	if len(cfg.ProdPatterns) != 2 || cfg.ProdPatterns[0] != "staging" {
		t.Errorf("ProdPatterns = %v, want [staging preprod]", cfg.ProdPatterns)
	}
	if len(cfg.ReadonlyNamespaces) != 2 || cfg.ReadonlyNamespaces[0] != "kube-system" {
		t.Errorf("ReadonlyNamespaces = %v", cfg.ReadonlyNamespaces)
	}
	if cfg.Cache.PodsTTL != 10*time.Second {
		t.Errorf("Cache.PodsTTL = %v, want 10s", cfg.Cache.PodsTTL)
	}
	if cfg.Cache.NamespacesTTL != 60*time.Second {
		t.Errorf("Cache.NamespacesTTL = %v, want 60s", cfg.Cache.NamespacesTTL)
	}
	if cfg.Exec.Shell != "/bin/bash" {
		t.Errorf("Exec.Shell = %q, want /bin/bash", cfg.Exec.Shell)
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("{{invalid yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfigFrom(cfgPath)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestIsReadonlyNamespace(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		patterns  []string
		want      bool
	}{
		{"exact match", "kube-system", []string{"kube-system"}, true},
		{"glob match", "openshift-monitoring", []string{"openshift-*"}, true},
		{"no match", "my-app", []string{"kube-system", "openshift-*"}, false},
		{"empty patterns", "anything", nil, false},
		{"empty namespace", "", []string{"kube-system"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsReadonlyNamespace(tt.namespace, tt.patterns)
			if got != tt.want {
				t.Errorf("IsReadonlyNamespace(%q, %v) = %v, want %v", tt.namespace, tt.patterns, got, tt.want)
			}
		})
	}
}

func TestSplitSegments(t *testing.T) {
	tests := []struct {
		input string
		want  int // number of segments
	}{
		{"my-app-prod", 3},
		{"app.prod.ns", 3},
		{"app_prod_ns", 3},
		{"app-v2.prod_env", 4},
		{"prod", 1},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			segments := splitSegments(tt.input)
			if len(segments) != tt.want {
				t.Errorf("splitSegments(%q) = %v (len %d), want len %d", tt.input, segments, len(segments), tt.want)
			}
		})
	}
}
