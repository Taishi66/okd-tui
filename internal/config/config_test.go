package config

import "testing"

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
