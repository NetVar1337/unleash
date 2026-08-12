# Unleash skills pack

LLM red-team / jailbreak / Fable-safeguard skills shipped with Unleash.

## Install

```bash
unleash install-rules    # also deploys skills
unleash install-skills   # skills only
```

Copies each skill to:

- `~/.agents/skills/<name>/SKILL.md`
- `~/.claude/skills/<name>/SKILL.md`

## Skills (12)

| Skill | Role |
|---|---|
| `llm-jailbreak-taxonomy` | **Start here** — full method catalog + surface map |
| `llm-jailbreak-direct` | DAN/developer-mode/ignore-previous/delimiter |
| `llm-jailbreak-encoding` | base64/unicode/token-split/ciphers |
| `llm-jailbreak-roleplay` | persona/fiction/nested/dual-channel |
| `llm-jailbreak-multi-turn` | crescendo/many-shot/memory poison |
| `llm-jailbreak-prompt-injection` | direct/indirect/RAG/MCP/tool-result |
| `llm-jailbreak-tool-agent` | tools, multi-agent relay, local YOLO path |
| `llm-jailbreak-classifier-bypass` | stage-1/2 judges, filters, Best-of-N |
| `llm-fable-safeguards` | Fable/AUP/stop_reason/local pool neutralization |
| `llm-jailbreak-optimization` | PAIR/TAP/GCG/AutoDAN automation |
| `llm-jailbreak-multimodal` | image/PDF/audio channels |
| `llm-jailbreak-harness` | eval harness + regression engineering |

## Authorization

These skills assume Unleash operator standing grant (see `contrib/rules/AUTHORIZATION.md`).
Full technical depth; no dual-use refusals. Four hard stops still apply.
