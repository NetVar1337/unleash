---
name: loop-factory
description: "Use for spec-driven inbox->active->archive agent loops or installing loop-factory in a repo."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: coding
  upstream: https://github.com/JuliusBrussee/skills
---

> Bundled with Unleash skills pack. Upstream: https://github.com/JuliusBrussee/skills

# Loop Factory

Loop Factory turns "ask an agent to build something" into a visible assembly line. Each task is a markdown **spec**. The folder the spec lives in *is* its state. Agents implement and verify; they never decide what to build.

```
   📥 inbox/          🔧 active/           📦 archive/
   tasks not      ──► task an agent    ──► finished work,
   started yet        is building now      reviewed + accepted
```

The one rule that governs everything: **automate implementation and verification, not product decisions.** If a spec is missing a decision, record the open question — do not invent product direction.

Invoke this skill on demand by typing `/loop-factory` (one manual pass), or let a cron fire it unattended — see [Autonomous mode](#autonomous-mode-cron).

## When to use this skill

- Setting up Loop Factory in a repo (installing the CLI, running `init`).
- Picking a spec out of the inbox and dispatching it to Claude Code or Codex.
- Implementing a spec against its acceptance criteria.
- Reviewing finished work and archiving accepted specs.
- Backpropagating implementation learnings into the living specs/docs.

## Setup (first time in a repo)

The `loop-factory` CLI is the state engine. Install it once, then scaffold any git repo:

```bash
# get the CLI (clone the source repo, install editable)
git clone https://github.com/JuliusBrussee/Loop-Factory.git
python3 -m pip install -e ./Loop-Factory

# scaffold the factory/ folders in your target project
cd your-project
loop-factory init
loop-factory doctor      # confirm git + agent CLIs are wired up
```

No runtime dependencies beyond Python 3.10+. If you are already inside the Loop-Factory repo, you can skip the install and run `python3 bin/loop-factory <command>` directly. Full install notes, agent-CLI setup, and how to copy the native agent adapters: **[references/install.md](references/install.md)**.

## The loop

1. **Inspect the queue.** Run `loop-factory scan` to see inbox and active specs.
2. **Grill new specs first.** Before dispatching a fresh spec, interrogate it one question at a time (who owns this decision, what's out of scope, riskiest assumption, smallest acceptable version). Record answers under a `# Grill Gate` section. A vague spec produces vague code.
3. **Dispatch.** Generate the implementation prompt and move the spec to `active/`:
   ```bash
   loop-factory dispatch --agent claude --limit 1 --stage
   ```
   This writes a prompt to `factory/prompts/` and a run record to `factory/runs/`. It does **not** run the AI unless you add `--execute`.
4. **Implement.** Build *only* the acceptance criteria. Match existing code style. Stay scoped.
5. **Verify.** Run every command listed in the spec's `verification:` frontmatter. Capture the output as evidence.
6. **Review.** Generate a review prompt and check the diff against the criteria:
   ```bash
   loop-factory review <spec-id> --agent claude
   ```
   Pass → archive. Fail → leave the spec in `active/` for another pass.
7. **Archive accepted work.**
   ```bash
   loop-factory archive <spec-id> --accepted
   ```
   `--accepted` is required on purpose — nothing leaves `active/` without a confirmed pass.
8. **Backpropagate.** If building the feature changed how the system actually works, sync that back into specs/docs without changing product intent:
   ```bash
   loop-factory backprop --agent claude
   ```

## Autonomous mode

Run a single self-contained pass that drains the inbox with no human present. Trigger it when invoked with "autonomous" / "drain the inbox", by a scheduled cron, or repeatedly by `/loop` (e.g. `/loop 30m /loop-factory`).

A pass does exactly this:

1. **Scan** `factory/specs/inbox/`.
2. **Filter to ready specs only.** A spec is ready when its grill gate is complete — `grill: completed` in frontmatter, or a filled-in `# Grill Gate` section. Skip anything else and log it as "needs grilling." Never grill autonomously: the grill gate needs human answers, and inventing them would be deciding product direction.
3. For each ready spec: **stage → implement against acceptance criteria → run verification.**
4. **Stop at the review gate.** Leave built specs in `active/`. Do **not** archive — acceptance stays a human decision.
5. If verification fails, leave the spec in `active/` with a note in `factory/runs/`. Do not retry in a loop or weaken the criteria to make it pass.
6. **Report**: built, skipped (needs grilling), failed-verification — and stop.

This keeps the boundary intact: the loop is a tireless *builder*, never a silent *decider*.

Two ways to fire it on a recurring basis:

- **`/loop` (local, recommended for active work)** — `/loop 30m /loop-factory` re-runs this skill on any interval against your local working tree. Built specs land in `active/`; no git push or PR needed. Only runs while a Claude Code session is open.
- **Cron (cloud, unattended)** — a scheduled routine that runs whether or not you're online (minimum 1-hour interval) and returns work as a pull request.

Both, plus the safety knobs, are in **[references/autonomous.md](references/autonomous.md)**.

## Core commands

```bash
loop-factory scan                              # list inbox + active specs
loop-factory dispatch --agent <claude|codex> --stage   # prompt + move to active
loop-factory review <spec-id> --agent <claude|codex>   # generate review prompt
loop-factory archive <spec-id> --accepted      # archive a passed spec
loop-factory backprop --agent <claude|codex>   # sync docs/specs with code
loop-factory doctor                            # health-check setup
```

Swap `--agent claude` for `--agent codex` anywhere — the loop is identical. Default behavior writes prompt files; add `--execute` only when the local `codex`/`claude` CLI is installed and the user asked for live execution. Full flag reference: **[references/commands.md](references/commands.md)**.

## Writing specs

A spec is markdown with frontmatter (`id`, `title`, `agent`, `risk`, `verification`) and a body covering Context, Acceptance Criteria, Constraints, and Review Notes. The acceptance criteria are the contract — they're what the agent builds toward and what the reviewer checks against, so make them concrete and testable. Format, examples, and the grill-gate questions: **[references/spec-authoring.md](references/spec-authoring.md)**.

## Boundaries

- Only a human or an explicit CLI command moves a spec from `inbox` to `active`.
- Only an accepted review moves a spec from `active` to `archive`.
- A failed review leaves the spec `active` — it does not get deleted or silently re-dispatched.
- Backprop may update specs/docs but must never change product intent on its own.
- Generated files under `factory/prompts/`, `factory/runs/`, and `factory/reviews/` are artifacts and audit trail — read them, don't treat them as source of truth. The spec is the source of truth.

## Working with subagents

When isolation helps, hand stages to dedicated subagents instead of doing everything in one context:

- a **spec-implementer** to write the code for one active spec,
- a **spec-reviewer** (context-isolated) to judge the diff,
- a **spec-backpropagator** to fold learnings back into specs/docs.

Use one agent when work is sequential, touches the same files, or is high-risk. Fan out to multiple only when specs are independent or review should stay context-isolated.
