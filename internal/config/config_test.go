package config

import "testing"

func TestIsProdNamespace(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		patterns  []string
		want      bool
	}{
		// Positive cases
		{"exact prod", "prod", nil, true},
		{"contains prod", "my-app-prod", nil, true},
		{"contains production", "production", nil, true},
		{"contains prd", "app-prd-01", nil, true},
		{"contains live", "live-env", nil, true},

		// Case insensitive
		{"uppercase PROD", "PRODUCTION", nil, true},
		{"mixed case", "My-Prod-Namespace", nil, true},

		// Negative cases
		{"dev namespace", "development", nil, false},
		{"staging", "staging", nil, false},
		{"test", "test-env", nil, false},
		{"empty namespace", "", nil, false},

		// Custom patterns
		{"custom pattern match", "my-staging", []string{"staging"}, true},
		{"custom pattern no match", "my-dev", []string{"staging"}, false},
		{"empty custom patterns uses defaults", "production", []string{}, true},

		// Edge cases
		{"product contains prod", "product-api", nil, true}, // BUG? "product" contains "prod"
		{"reproduce contains prod", "reproduce-bug", nil, true}, // BUG? substring match too broad
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
	// Empty slice should use defaults, not match nothing
	if !IsProdNamespace("production", []string{}) {
		t.Error("empty patterns should use defaults, but didn't match 'production'")
	}
}
