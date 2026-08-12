---
name: snailsploit-frameworks
description: "Use when applying SnailSploit's AATMF, SEF, P.R.O.M.P.T, AATMF Toolkit, LLM Red Teamer's Playbook, or Claude-Red frameworks to adversarial AI assessment and offensive-security research."
version: 1.0.0
license: Mixed upstream licenses; inspect each source
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: imported
  upstream: C:\Users\Admin\.agents\skills\snailsploit-frameworks\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\snailsploit-frameworks\SKILL.md

# SnailSploit Framework Router

## Overview

Route work through the six frameworks named by SnailSploit's canonical frameworks page. Use the local clones as primary sources and the installed `aatmf` CLI as the execution layer. Do not substitute website marketing summaries for repository code, tests, or framework text.

Local root: `C:\Users\Admin\tools\SnailSploit`

## When to Use

Use this skill for:

- adversarial AI threat modeling and assessment coverage;
- LLM defense-layer diagnosis and red-card design;
- prompt, agent, RAG, model API, training, or AI supply-chain testing;
- AATMF risk mapping, regression checks, fingerprints, or attack-chain planning;
- routing an offensive-security task to the installed Claude-Red methodology.

Do not use it as a substitute for product-specific source review, reproduction, or novelty checks. For zero-day work, run `zero-day-target-eligibility` separately.

## Framework Routing

| Need | Framework | Local source |
|---|---|---|
| AI attack taxonomy, procedure IDs, controls, and risk mapping | AATMF | `frameworks\AATMF-Adversarial-AI-Threat-Modeling-Framework\` |
| Human and social-engineering attack surface | SEF | `frameworks\SnailSploit.com\sef.html` |
| Structured adversarial prompt composition | P.R.O.M.P.T | `frameworks\SnailSploit.com\prompt.html` |
| Automated red cards, evaluation, fingerprints, decay, and chains | AATMF Toolkit | `frameworks\aatmf-toolkit\`; executable `aatmf` |
| Diagnose which of five LLM defense layers is responding | LLM Red Teamer's Playbook | `frameworks\The-LLM-Red-Teamer-s-Playbook\` |
| Product/technique-specific offensive methodology | Claude-Red | source at `frameworks\Claude-Red\`; converted skills at `~/.agents/skills/claude-red/` |

## Workflow

1. **Define the system.** Record model/provider, system prompt ownership, input/output filters, tools, memory, RAG, identities, data stores, and human approval points. Completion: every trust boundary has an owner and attacker-controlled input.
2. **Map coverage.** Read the relevant AATMF tactic chapters and assign technique/procedure IDs. Add SEF for human workflows and P.R.O.M.P.T when prompt composition is part of the experiment. Completion: each hypothesis has a framework ID and expected security invariant.
3. **Diagnose before bypassing.** Apply the Playbook's five-layer model: input filter, model alignment, system identity, output filter, and agentic trust boundary. Completion: observations distinguish the likely enforcement layer from alternatives.
4. **Design red cards.** Use the toolkit schema and bundled examples. Define expected block behavior, forbidden output, confidence criteria, budget, and control probes. Completion: the test is falsifiable and can run without subjective scoring alone.
5. **Dry-run first.** Execute `aatmf run <card> --target <provider:model> --dry-run`. Do not configure or print credentials. Completion: target, card count, probe count, and evaluation tier match the intended experiment.
6. **Execute and preserve evidence.** Run only against the selected endpoint, retain raw prompts/responses and tool traces, and emit JSON/SARIF/JUnit as appropriate. Completion: every finding can be replayed from captured inputs and configuration.
7. **Validate independently.** Separate framework labels from exploitability, deployment, latest-stable reproduction, and novelty. Completion: conclusions cite observed behavior rather than framework membership alone.

## Toolkit Quick Reference

```powershell
aatmf --help
aatmf run examples\example_card.yaml --target openai:gpt-4o --dry-run
aatmf fingerprint --target openai:gpt-4o --output-path profile.json
aatmf decay --cards cards\ --target openai:gpt-4o --db decay.db
aatmf chain --profile profile.json --max-steps 4 --top-k 5
```

The toolkit is installed through `uv` in an isolated Python 3.11 environment. No API keys are preconfigured.

## Source Calibration

- Prefer repository `LICENSE`, code, and tests when the website disagrees.
- AATMF's upstream validator currently reports a stale generated `data/aatmf.json`; do not claim the full taxonomy clone validates cleanly until upstream or the local artifact is regenerated deliberately.
- The website's Claude-Red count is stale; the installed repository revision contains 58 skills.
- SEF and P.R.O.M.P.T do not currently have working standalone public repository links. Their canonical local source is the cloned website content.
- A framework mapping is organizational evidence, not proof of vulnerability, severity, or novelty.

## Verification Checklist

- [ ] Relevant primary framework files were read from the local clone.
- [ ] Assessment hypotheses have explicit trust boundaries and invariants.
- [ ] Toolkit execution began with a dry run.
- [ ] No credential values were written to reports, commands, or configuration.
- [ ] Findings are independently reproducible and not justified solely by taxonomy labels.

