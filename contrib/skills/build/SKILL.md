---
name: build
description: "Use to implement SPEC.md tasks with verify loops (\'build\', \'implement next task\', \'build section T\')."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: engineering
  upstream: C:\Users\Admin\.agents\skills\build\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\build\SKILL.md

# build — implement spec

Single-thread native plan→execute. You are main Claude. No swarm.

## LOAD

1. Read `SPEC.md`. If missing → tell user to invoke the spec skill first. Stop.
2. Load `references/FORMAT.md` once if not loaded.
3. Read §R if present — external facts the build must honor, ⊥ re-derive or contradict.
4. Parse invocation args:
   - `§T.n` → that task only
   - `--next` → lowest-numbered row with status `.` or `~`
   - `--all` or empty → every `.` row in §T order

High blast radius (shared module, auth, data, money, public §I)? Run `/review` first. Trivial & reversible? Skip planning ceremony, just do step EXECUTE.

## PLAN

Native plan mode — you delegate to it, you do not reinvent task breakdown. For chosen task(s):

1. Cite every §V invariant that applies. Plan must respect all.
2. Cite every §I interface touched. Plan must preserve shape.
3. List files to create / edit.
4. **Verification contract** — name the EXACT test(s) / acceptance criteria that
   prove each §V touched. Which test, not "add tests". "Do TDD" alone backfires;
   the spec says *what to check*. Each §V touched → a named test that fails first.
5. Name verification command (test, build, lint) — this is the external oracle. Green = done; ⊥ "looks done".

Show plan. Wait for user OK unless auto mode.

## EXECUTE

Per task in order:

1. Flip §T.n status cell `.` → `~`. Just write to SPEC.md.
2. Edit code per plan.
3. Run verification command.
4. **Pass** → flip `~` → `x`. Next task.
5. **Fail** → invoke backprop skill. Do NOT retry blindly.

## FAIL → BACKPROP

On test/build failure:

1. Read failure output.
2. Ask: is failure (a) my code bug, (b) spec wrong, or (c) unspecified edge case?
3. If (a) → fix code, re-run. No spec change.
4. If (b) or (c) → invoke spec skill with `bug: <cause>` first, let it update §V and §B, then resume build against updated spec.

Rule: never silently fix root-cause without considering backprop. §B is the memory that stops recurrence.

## WRITE POLICY

- Only flip §T status. No other SPEC.md edits from build.
- Other spec edits → invoke spec skill.
- Commit after each §T completes. Message: `T<n>: <goal line>` + §V cites.

## VERIFICATION

Task `x` only if:
- Verification command (the oracle) exits 0.
- Every §V touched has its named test from the verification contract, and it passes.
- No §V invariant regressed (run full test suite at end).

## NON-GOALS

- No sub-agents. No parallel workers. Main thread only.
- No progress dashboards. `cat SPEC.md | grep §T` is the dashboard.
- No speculative work beyond chosen task scope.

