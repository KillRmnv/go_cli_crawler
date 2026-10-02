package clicrawler

import (
	"testing"
)

func TestHostInScope(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		site     string
		expected bool
	}{
		{"Exact match", "example.com", "example.com", true},
		{"Same host different case", "EXAMPLE.COM", "example.com", true},
		{"Trailing dot in host", "example.com.", "example.com", true},
		{"Port stripped on both sides", "example.com:8080", "example.com:9090", true},
		{"Subdomain is in scope", "blog.example.com", "example.com", true},
		{"Www subdomain is in scope", "www.example.com", "example.com", true},
		{"Deep subdomain is in scope", "a.b.example.com", "example.com", true},

		{"Prefix lookalike is out of scope", "myexample.com", "example.com", false},
		{"Suffix lookalike is out of scope", "notexample.com", "example.com", false},
		{"Lookalike as subdomain is out of scope", "example.com.evil.com", "example.com", false},
		{"Lookalike as parent is out of scope", "evil-example.com", "example.com", false},
		{"Other domain is out of scope", "other-domain.com", "example.com", false},

		{"IP exact match", "127.0.0.1", "127.0.0.1", true},
		{"IP with port", "127.0.0.1:8791", "127.0.0.1:8080", true},
		{"IP lookalike is out of scope", "127.0.0.1.evil.com", "127.0.0.1", false},
		{"IP suffix boundary is respected", "0.1", "127.0.0.1", false},
		{"IPv6 exact match", "[::1]", "[::1]:8080", true},
		{"IPv6 loopback vs IPv4 loopback", "::1", "127.0.0.1", false},

		{"Single label site matches only itself", "com", "com", true},
		{"Single label site rejects subdomains", "example.com", "com", false},
		{"Empty host is rejected", "", "example.com", false},
		{"Empty site is rejected", "example.com", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := hostInScope(tc.host, tc.site); got != tc.expected {
				t.Errorf("hostInScope(%q, %q) = %v, expected %v", tc.host, tc.site, got, tc.expected)
			}
		})
	}
}

func TestUrl_SiteHost(t *testing.T) {
	tests := []struct {
		name        string
		adress      string
		expected    string
		expectError bool
	}{
		{"Plain host", "https://example.com/page", "example.com", false},
		{"Host with path and port", "http://example.com:8080/path?a=b", "example.com", false},
		{"Upper case host", "https://EXAMPLE.com/", "example.com", false},
		{"Trailing dot", "https://example.com./", "example.com", false},
		{"IPv6 literal", "http://[::1]:8080/", "::1", false},
		{"IP literal", "http://127.0.0.1:8791/", "127.0.0.1", false},

		{"Empty url", "", "", true},
		{"No scheme", "example.com", "", true},
		{"No host", "https:///page", "", true},
		{"Not a url", "://", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u := NewUrl(tc.adress)
			host, err := u.SiteHost()
			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error for %q, recieved host %q", tc.adress, host)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.adress, err)
			}
			if host != tc.expected {
				t.Errorf("expected host %q, recieved %q", tc.expected, host)
			}
		})
	}
}
