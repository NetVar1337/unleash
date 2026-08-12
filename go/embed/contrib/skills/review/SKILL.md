---
name: review
description: "Use before high-blast-radius builds to adversarially review SPEC.md (\'review the spec\', /ck:review)."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: engineering
  upstream: C:\Users\Admin\.agents\skills\review\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\review\SKILL.md

# review — refute the spec before build

**Every finding cites evidence — file:line or a source. No evidence → flag `[unverified]`. Default to refuted: a flaw you cannot prove is a flaw you note, not one you wave through.**

An LLM cannot self-correct on its own judgment — left alone it drifts or
degrades. Review fixes that the only way that works: a *separate* skeptic
anchored to an *external oracle* — the code, §R, the test suite, the docs.
"Looks good" is not a review. A refutation attempt is.

## WHEN TO REVIEW

- Before `/build` on a high-blast-radius change (shared module, auth, data, money, public API).
- Spec touched §I or §V that other code depends on.
- Right-sizing says the cost of a wrong build > the cost of one review pass.

Skip for a trivial, reversible, well-understood change. Adversarial review on a
typo hallucinates flaws & wastes the budget — the self-critique paradox is real.

## PHASE 0 — CAPTURE

Read the spec: §G §C §I §R §V §T. Hold the whole thing. You review the *spec*,
not your memory of the conversation.

## PHASE 1 — CONSTRUCT THE SENIOR

Build a reviewer with real authority, not a generic critic:
- **Codebase** — grep/read the modules this spec touches. What patterns, what invariants already hold?
- **§R** — what did research establish? A spec decision that contradicts §R is a finding.
- **Live** — for any best-practice claim you are unsure of, fetch it. An out-of-date assumption is a flaw.

A reviewer with no evidence is just an opinion. Earn the authority first.

## PHASE 2 — REFUTE

Attack the spec on these axes. For each, try to find the case where it breaks:
- **Goal vs reality** — does §G solve the actual problem, or a proxy?
- **Missing invariant** — what can go wrong that no §V catches? (most findings live here)
- **Interface drift** — does §I match what callers already expect? (cite the caller, file:line)
- **Constraint conflict** — do two §C bullets contradict? does one fight §R?
- **Unowned edge** — the input, ordering, failure, or concurrency case no §T covers.
- **Altitude** — §T too vague to act on, or so granular it is just typing?

## PHASE 3 — CLASSIFY

Each finding: `evidence → claim → severity`.
- **BLOCK** — build on this spec ships a real defect. Must fix first.
- **HARDEN** — add/sharpen a §V so the build cannot regress it.
- **NOTE** — worth knowing, not blocking.

No evidence? Down-rank to NOTE & tag `[unverified]`. ⊥ inflate a hunch to BLOCK.

## PHASE 4 — HARDEN §V & GATE

- Each HARDEN finding → a draft §V line (testable, cites the §I/behavior it guards). Hand to **spec** to write.
- End on an explicit gate:

```
## review verdict
BLOCK: 1 — §I.api shape ≠ caller src/client.ts:40. fix §I before build.
HARDEN: 2 — drafted V8 (idempotent refund), V9 (tx around dual write).
NOTE: 1 — §T4 vague, split before /build.
gate: NO-GO until BLOCK cleared. then /build §T after spec writes V8,V9.
```

GO or NO-GO, never a shrug. Review is the checkpoint that stops a confident wrong build.

## BOUNDARIES

- ⊥ write SPEC.md. Draft §V & hand to spec.
- ⊥ pass a finding with no evidence as fact. Flag `[unverified]`.
- ⊥ review trivia. Right-size or skip.
- ⊥ rewrite the user's intent. You harden the spec, you do not replace its goal.

