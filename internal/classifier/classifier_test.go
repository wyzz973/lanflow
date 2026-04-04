package classifier

import (
	"testing"
)

func TestEmbeddedRulesLoaded(t *testing.T) {
	c := New(nil)

	tests := []struct {
		domain string
		want   string
	}{
		{"www.bilibili.com", "B站"},
		{"github.com", "GitHub"},
		{"api.github.com", "GitHub"},
		{"www.baidu.com", "百度"},
		{"www.google.com", "Google"},
		{"unknown-random-site.xyz", ""},
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
	if got := c.Classify("GitHub.COM"); got != "GitHub" {
		t.Errorf("Classify(GitHub.COM) = %q, want GitHub", got)
	}
}

func TestRuleCount(t *testing.T) {
	c := New(nil)
	total := len(c.suffixes) + len(c.exacts)
	if total < 1000 {
		t.Errorf("expected at least 1000 rules, got %d", total)
	}
	t.Logf("loaded %d rules (%d suffix, %d exact)", total, len(c.suffixes), len(c.exacts))
}
