package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// www removal
		{"https://www.example.org", "https://example.org"},
		// trailing slash removal
		{"https://example.org/path/", "https://example.org/path"},
		// host lowercased
		{"https://EXAMPLE.ORG/path", "https://example.org/path"},
		// scheme lowercased
		{"HTTPS://example.org/path", "https://example.org/path"},
		// default port removal
		{"https://example.org:443/path", "https://example.org/path"},
		// dot segments resolved
		{"https://example.org/a/../b", "https://example.org/b"},
		// www + trailing slash combined
		{"https://www.example.org/repo/", "https://example.org/repo"},
		// already normalized: no change
		{"https://example.org/repo", "https://example.org/repo"},
		// invalid URL: returned as-is
		{"not-a-valid-url", "not-a-valid-url"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, NormalizeURL(tt.input))
		})
	}
}
