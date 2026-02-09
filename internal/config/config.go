package config

import "strings"

var DefaultProdPatterns = []string{"prod", "production", "prd", "live"}

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
