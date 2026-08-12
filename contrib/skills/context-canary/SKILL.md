---
name: context-canary
description: "Use to install a per-turn canary against context rot/compaction (\'canary\', \'did you lose context\')."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: coding
  upstream: https://github.com/JuliusBrussee/skills
---

> Bundled with Unleash skills pack. Upstream: https://github.com/JuliusBrussee/skills

# Context Canary

A context canary is a trivially checkable standing instruction whose only job is to fail visibly. The classic form: "start every response with my name." The agent costs almost nothing to comply — so when the name disappears, that absence is data. The instruction didn't get harder; the agent's hold on its early-context instructions got weaker. Like the coal-mine canary, it dies first, before the failures you actually care about (forgotten constraints, ignored conventions, re-litigated decisions) start landing in your code.

This works because context degradation is **silent and gradual**. Models don't announce that they've stopped attending to instructions from 80k tokens ago, and compaction summaries quietly drop standing instructions. The canary converts an invisible failure into a binary, per-turn, zero-effort check. The research behind why this happens — context rot, lost-in-the-middle, instruction drift, compaction loss — is in **[references/research.md](references/research.md)**.

## When to use this skill

- Installing a canary at the start of a long or high-stakes session.
- The user notices the canary stopped appearing and asks what happened.
- The agent itself realizes it can no longer find its canary contract in context.
- The user asks how trustworthy the current context still is.

## The canary contract

When invoked, install the canary by stating the contract explicitly in one short message, then follow it. Default format — first line of every response from then on:

```
**Julius · t14 · ctx ok**
```

Three fields, each probing something different:

| Field | Example | What its failure means |
|---|---|---|
| Name | `Julius` | The standing instruction itself fell out of effective context — adherence drift or compaction dropped it. |
| Turn counter | `t14` | Increment by 1 every response. A reset, skip, or repeat means continuity broke — almost always compaction or a summarization boundary. |
| Self-check | `ctx ok` / `ctx aging` / `ctx thin` | The agent's honest estimate. `aging`: the session is long and early details are getting summarized in its own working sense of the task. `thin`: the agent is reconstructing earlier decisions instead of remembering them. |

Use the user's actual name (ask if unknown). If the user wants the minimal version, name-only is fine — it's the original trick and still catches the big failures. Never explain, apologize for, or decorate the canary line; it must stay byte-stable so a human can pattern-match it in half a second.

**Session canary vs. standing canary.** A canary that lives only in the conversation tests whether *conversation* context survives (compaction, truncation, drift). A canary written into `CLAUDE.md` or agent memory survives compaction by design — which means it tests whether file-based instructions are being attended to, a different and weaker signal. Default to the session canary; that's the one that detects degradation. Offer the standing variant only if the user wants the habit across all sessions, and tell them what it no longer measures.

## Emission rules

1. Canary is the **first line** of every response, including short ones, error reports, and responses after tool calls.
2. Increment the counter every response. If unsure of the count, that uncertainty IS a signal — emit `t?` and flag it, never guess a plausible number.
3. The self-check must be honest. Reporting `ctx ok` by reflex defeats the entire instrument.
4. If at any point the canary contract cannot be found in context (you only know about it from a summary, or not at all), **declare a trip yourself** — don't wait for the user to notice.

## Trip protocol

A trip is: the canary missing, malformed, a counter discontinuity, or an agent-side self-declaration.

Calibrate before alarming — one missed canary on an otherwise coherent response is a warning (note it, resume the canary, keep going). **Two consecutive misses, a counter discontinuity, or the agent failing its own contract check is a confirmed trip.** Then:

1. **Stop trusting drifted state.** Do not barrel ahead on the current task using context you can no longer vouch for.
2. **Checkpoint.** Write the durable state somewhere outside the conversation: current goal, decisions made and why, files touched, what's verified vs. in-progress, next step. A `HANDOFF.md` or notes file in the repo works.
3. **Re-anchor.** Re-read the project instructions (`CLAUDE.md`, the original task statement) and the checkpoint. State back to the user, in three or four lines, what you believe the task and constraints are — so they can correct any drift cheaply.
4. **Reset deliberately.** Recommend the user start a fresh session (or run an explicit compact) seeded with the checkpoint file, rather than limping on. Degraded context doesn't heal; it compounds.
5. **Re-install the canary** with the counter reset to `t1`, noting the generation: `t1 (gen 2)`.

Never silently resume the canary after a gap as if nothing happened — that destroys the instrument's credibility, which is all it has.

## What the canary does and doesn't tell you

The test is one-sided. A **missing** canary is strong evidence of degradation. A **present** canary is weak evidence of health — a cheap stylistic habit can survive while harder constraints (architecture decisions, "never touch X") quietly rot, and a canary stuffed into memory files can outlive the context it was meant to monitor. Treat the canary as a smoke detector, not a structural inspection: when it fires, act; when it's quiet, stay reasonably suspicious in sessions past roughly 50% of the context window or after any compaction event.

Pair it with the cheap structural habits that reduce what the canary has to catch: keep durable decisions in files instead of chat, compact at deliberate boundaries instead of waiting for forced compaction, and prefer fresh sessions per task over one immortal session. Details and sources in **[references/research.md](references/research.md)**.
