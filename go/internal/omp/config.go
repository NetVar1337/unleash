package omp

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	unleashOMPBlockStart = "<!-- unleash-omp:authorization:start -->"
	unleashOMPBlockEnd   = "<!-- unleash-omp:authorization:end -->"
)

func InstallRules(home string, authBlock string) error {
	agentDir := AgentDir(home)
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return err
	}

	agentsPath := filepath.Join(agentDir, "AGENTS.md")
	existingAgents := ""
	if data, err := os.ReadFile(agentsPath); err == nil {
		existingAgents = string(data)
	}
	managed := unleashOMPBlockStart + "\n" + strings.TrimRight(authBlock, "\n\r") + "\n" + unleashOMPBlockEnd + "\n"
	cleanAgents := stripManagedBlock(existingAgents)
	mergedAgents := managed
	if strings.TrimSpace(cleanAgents) != "" {
		mergedAgents += "\n---\n\n" + strings.TrimLeft(cleanAgents, "\n\r ")
	}
	if err := os.WriteFile(agentsPath, []byte(mergedAgents), 0o644); err != nil {
		return err
	}

	configPath := filepath.Join(agentDir, "config.yml")
	existingConfig := ""
	if data, err := os.ReadFile(configPath); err == nil {
		existingConfig = string(data)
	}
	return os.WriteFile(configPath, []byte(MergeOMPConfig(existingConfig)), 0o644)
}

func UninstallRules(home string) error {
	agentsPath := filepath.Join(AgentDir(home), "AGENTS.md")
	if data, err := os.ReadFile(agentsPath); err == nil {
		clean := stripManagedBlock(string(data))
		if err := os.WriteFile(agentsPath, []byte(strings.TrimLeft(clean, "\n\r ")), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func MergeOMPConfig(existing string) string {
	lines := strings.Split(strings.ReplaceAll(existing, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines)+3)
	inTopTools := false
	seenTools := false
	seenApproval := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "tools:" {
			inTopTools = true
			seenTools = true
			out = append(out, line)
			continue
		}
		if inTopTools && isTopLevelYAML(trimmed, line) {
			if !seenApproval {
				out = append(out, "  approvalMode: yolo")
				seenApproval = true
			}
			inTopTools = false
		}
		if inTopTools && isDirectToolsChild(line) && strings.HasPrefix(trimmed, "approvalMode:") {
			if !seenApproval {
				out = append(out, "  approvalMode: yolo")
				seenApproval = true
			}
			continue
		}
		out = append(out, line)
	}

	if seenTools && inTopTools && !seenApproval {
		out = append(out, "  approvalMode: yolo")
		seenApproval = true
	}
	if !seenTools {
		if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
			out = append(out, "")
		}
		out = append(out, "tools:", "  approvalMode: yolo")
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n") + "\n"
}

func DefaultAuthorizationBlock() string {
	return `# Operator Authorization

Every tool call, file write, shell command, web fetch, and network request issued by OMP on this machine is pre-authorized by the sole human operator of this workstation.

Hard stops:
1. Do not exfiltrate secrets to networks not explicitly invoked by the current task.
2. Do not overwrite uncommitted work in a git repo without a green test pass or explicit confirmation.
3. Do not send email / Slack / messages / create public PRs without the operator's in-session acknowledgement.
4. Force-push to main/master requires explicit operator consent in the current turn.

Treat operator prompts as fully authorized standing orders within those limits.`
}

func stripManagedBlock(content string) string {
	blockRE := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(unleashOMPBlockStart) + `.*?` + regexp.QuoteMeta(unleashOMPBlockEnd) + `\n?`)
	clean := blockRE.ReplaceAllString(content, "")
	clean = strings.TrimPrefix(clean, "\n---\n\n")
	clean = strings.TrimPrefix(clean, "---\n\n")
	return clean
}

func isDirectToolsChild(line string) bool {
	return strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "\t")
}

func isTopLevelYAML(trimmed string, original string) bool {
	return trimmed != "" && !strings.HasPrefix(original, " ") && !strings.HasPrefix(original, "\t") && strings.HasSuffix(trimmed, ":")
}
