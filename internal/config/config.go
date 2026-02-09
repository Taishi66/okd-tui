package config

import "strings"

var DefaultProdPatterns = []string{"prod", "production", "prd", "live"}

func IsProdNamespace(namespace string, patterns []string) bool {
	if len(patterns) == 0 {
		patterns = DefaultProdPatterns
	}
	ns := strings.ToLower(namespace)
	for _, p := range patterns {
		if strings.Contains(ns, strings.ToLower(p)) {
			return true
		}
	}
	return false
}
