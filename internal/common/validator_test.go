package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateCodeHostingURL(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		valid bool
	}{
		{"https github", "https://github.com/foo/bar", true},
		{"http public", "http://example.org/repo", true},
		{"empty", "", true},
		{"userinfo with password", "https://user:pass@example.org/repo", true},
		{"userinfo no password", "https://user@example.org/repo", true},

		{"ftp scheme", "ftp://example.org/repo", false},
		{"file scheme", "file:///etc/passwd", false},
		{"git scheme", "git://example.org/repo.git", false},

		{"localhost host", "https://localhost/repo", false},
		{"localhost trailing dot", "https://localhost./repo", false},
		{"LOCALHOST trailing dot", "https://LOCALHOST./repo", false},
		{"loopback ipv4", "https://127.0.0.1/repo", false},
		{"loopback ipv6", "https://[::1]/repo", false},
		{"unspecified", "https://0.0.0.0/repo", false},

		{"private 10/8", "https://10.0.0.1/repo", false},
		{"private 172.16/12", "https://172.16.0.1/repo", false},
		{"private 192.168/16", "https://192.168.1.1/repo", false},
		{"link local v4", "https://169.254.1.1/repo", false},
		{"link local v6", "https://[fe80::1]/repo", false},

		{"missing host", "https:///repo", false},
		{"malformed", "::not a url::", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := struct {
				URL string `validate:"omitempty,http_url,code_hosting_url"`
			}{URL: tt.url}

			errs := ValidateStruct(payload)

			if tt.valid {
				assert.Empty(t, errs, "expected %q to validate", tt.url)
			} else {
				assert.NotEmpty(t, errs, "expected %q to fail", tt.url)
			}
		})
	}
}
