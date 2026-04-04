package classifier

import (
	"testing"
)

func TestClassify(t *testing.T) {
	c := New(nil)

	// Manually add some rules for testing
	c.suffixes["bilibili.com"] = "B站"
	c.suffixes["github.com"] = "GitHub"
	c.exacts["api.google.com"] = "Google"
	c.suffixes["google.com"] = "Google"

	tests := []struct {
		domain string
		want   string
	}{
		{"www.bilibili.com", "B站"},
		{"bilibili.com", "B站"},
		{"video.bilibili.com", "B站"},
		{"github.com", "GitHub"},
		{"api.github.com", "GitHub"},
		{"api.google.com", "Google"},   // exact match
		{"maps.google.com", "Google"},  // suffix match
		{"unknown-site.xyz", ""},       // no match
	}

	for _, tt := range tests {
		got := c.Classify(tt.domain)
		if got != tt.want {
			t.Errorf("Classify(%q) = %q, want %q", tt.domain, got, tt.want)
		}
	}
}

func TestClassifyCaseInsensitive(t *testing.T) {
	c := New(nil)
	c.suffixes["github.com"] = "GitHub"

	if got := c.Classify("GitHub.COM"); got != "GitHub" {
		t.Errorf("Classify(GitHub.COM) = %q, want GitHub", got)
	}
}
