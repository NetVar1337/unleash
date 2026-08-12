---
name: llm-jailbreak-harness
description: "Build jailbreak/red-team harnesses: datasets, scorers, runners, logging, Unleash integration, regression after model updates."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  category: llm-redteam
  author: NetVar1337/unleash
  triggers:
    - "jailbreak harness"
    - "red team eval"
    - "garak"
    - "prompt regression"
---
# Jailbreak harness engineering

## Minimal harness
- `seeds/` prompts by taxonomy tag
- `runners/target.py` model/agent adapter
- `score.py` rubric (refuse/partial/full + keyword checks)
- `results/` jsonl transcripts
- `report.md` generator

## Good scorers
- Refusal detector (string + judge)
- Technical completeness checklist per task type
- Safety-filter hit taxonomy
- Latency/cost

## CI ideas
- Nightly against local agent builds
- Diff refusal strings after Claude/Codex updates → feed Unleash re-anchor
- Track regression: yesterday's wins failing today

## Unleash integration
- On local CC/Codex/OMP: setup → patch → verify before prompt campaigns
- Store new refusal strings under research notes for pool patches
- Skills pack path: `contrib/skills/` installed to `~/.agents/skills` + `~/.claude/skills`
