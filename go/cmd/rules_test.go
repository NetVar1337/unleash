package cmd

import (
	"encoding/json"
	"testing"
)

func TestSettingsRulesDoNotUseInvalidMCPWildcard(t *testing.T) {
	data, err := getContribFile("rules/settings-rules.json")
	if err != nil {
		t.Fatalf("load settings rules: %v", err)
	}

	var settings struct {
		Permissions struct {
			Allow []string `json:"allow"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("parse settings rules: %v", err)
	}

	for _, rule := range settings.Permissions.Allow {
		if rule == "mcp__*" {
			t.Fatalf("settings rules contain invalid Claude permission allow rule %q; use explicit mcp__<server>__* rules or rely on PreToolUse auto-allow", rule)
		}
	}
}
