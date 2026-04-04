package classifier

import (
	"bufio"
	_ "embed"
	"log/slog"
	"strings"
)

//go:embed rules.txt
var rulesData string

type Classifier struct {
	suffixes map[string]string // domain suffix → service name
	exacts   map[string]string // full:domain → service name
	logger   *slog.Logger
}

func New(logger *slog.Logger) *Classifier {
	c := &Classifier{
		suffixes: make(map[string]string),
		exacts:   make(map[string]string),
		logger:   logger,
	}
	c.loadEmbedded()
	return c
}

// isGenericService returns true for meta-category services whose domains
// should not override a more specific service classification.
func isGenericService(fileKey string) bool {
	return strings.HasPrefix(fileKey, "category-") ||
		strings.HasPrefix(fileKey, "geolocation-") ||
		fileKey == "tld-cn" || fileKey == "tld-not-cn" ||
		fileKey == "cn"
}

func (c *Classifier) loadEmbedded() {
	// Track the source file key for each domain so specific services
	// take priority over generic category/geolocation aggregates.
	suffixSource := make(map[string]string) // domain → file key
	exactSource := make(map[string]string)

	var currentService string
	var currentFileKey string
	scanner := bufio.NewScanner(strings.NewReader(rulesData))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			// Service header: "# bilibili:B站"
			header := line[2:]
			if idx := strings.Index(header, ":"); idx != -1 {
				currentFileKey = header[:idx]
				currentService = header[idx+1:]
			}
			continue
		}
		if currentService == "" {
			continue
		}

		if strings.HasPrefix(line, "full:") {
			domain := strings.ToLower(strings.TrimPrefix(line, "full:"))
			prev, exists := exactSource[domain]
			if !exists || (isGenericService(prev) && !isGenericService(currentFileKey)) {
				c.exacts[domain] = currentService
				exactSource[domain] = currentFileKey
			}
		} else {
			domain := strings.ToLower(line)
			prev, exists := suffixSource[domain]
			if !exists || (isGenericService(prev) && !isGenericService(currentFileKey)) {
				c.suffixes[domain] = currentService
				suffixSource[domain] = currentFileKey
			}
		}
	}

	if c.logger != nil {
		c.logger.Info("domain classifier loaded", "rules", len(c.suffixes)+len(c.exacts))
	}
}

// Classify returns the friendly service name for a domain, or empty string if unknown.
func (c *Classifier) Classify(domain string) string {
	domain = strings.ToLower(domain)

	// Try exact match first
	if name, ok := c.exacts[domain]; ok {
		return name
	}

	// Try suffix match
	d := domain
	for {
		if name, ok := c.suffixes[d]; ok {
			return name
		}
		idx := strings.Index(d, ".")
		if idx == -1 {
			break
		}
		d = d[idx+1:]
	}

	return ""
}
