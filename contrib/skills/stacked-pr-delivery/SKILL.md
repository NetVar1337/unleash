---
name: stacked-pr-delivery
description: "Use to advance/validate/merge stacked GitHub PR series safely."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: github
  upstream: C:\Users\Admin\.agents\skills\stacked-pr-delivery\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\stacked-pr-delivery\SKILL.md

# Stacked PR delivery

## Trigger
Use for a PR series where one PR targets another PR branch, especially after the base is squash-merged or when CI/policy state blocks an otherwise clean merge.

## Goal
Land independent PRs and the stack in dependency order without duplicating already-merged commits, bypassing required review, or trusting stale GitHub merge state.

## Procedure

1. **Map the graph before edits.** For every open PR, record draft state, base/head refs and SHAs, mergeability, required checks, review decision, and requested reviewers. Treat `main → A → B` as a dependency chain, not independent PRs.
2. **Check policy before expensive work.** Inspect required reviewers/code owners early. A code-owner requirement is a hard stop: do not use `--admin` to bypass it. Request the designated owner if it is not already requested, then continue only with non-merge preparation.
3. **Merge independent green PRs first.** Re-query their checks and merge state immediately before merging. Do not delete a merged branch that is still the base of a downstream PR.
4. **After a squash merge, rebase each child.** The merged squash SHA is not an ancestor of the old child branch. Rebase the child with `--onto origin/main <old-parent-head>`, validate locally, then force-push only with an explicit `--force-with-lease` expected SHA.
5. **Retarget only after the rebase.** Change the PR base to `main` through the REST pull-request endpoint when `gh pr edit --base` is affected by unrelated GraphQL/project-card failures. Confirm the GitHub base/head SHAs afterward.
6. **Make CI run on the current head.** Changing draft/base status may not fire workflows limited to `pull_request` open/sync/reopen. If no check run is created after confirming the workflow trigger, add a clearly named empty commit solely to emit a `synchronize` event; it will disappear in a squash merge.
7. **Resolve generated outputs from source.** For a conflict in a generated file, do not choose `ours` or `theirs` by guesswork. Run its repository generator against the pinned input/data bundle, then run its deterministic `--check` gate.
8. **Treat audit ignores as exceptions, not fixes.** First ask the resolver to upgrade the vulnerable package. If another required dependency makes that impossible, add the smallest documented advisory-specific ignore with the blocking constraint and a removal condition; verify the audit passes with that exact ignore.
9. **Merge only after policy and checks are green.** If GitHub still reports blocked, read the specific API/CLI reason. Do not retry merge commands blindly. Move to the next independently actionable task while waiting for human approval.

## Verification

- `git diff --check` is clean after every rebase/conflict resolution.
- Focused tests cover the changed implementation; run the repository CI mirror for dependency or generated-output changes.
- The current PR head—not an earlier SHA—has successful required checks.
- GitHub reports expected base/head refs, non-draft state, and no unresolved policy requirement before merge.
- After each merge, refresh all downstream PR topology before proceeding.

## Pitfalls

- A clean diff and successful CI do not satisfy code-owner review.
- `mergeable=MERGEABLE` can coexist with `mergeStateStatus=BLOCKED`; inspect the policy message instead of assuming a retry will help.
- Do not silently turn a source conflict into a generated-file hand edit.
- Do not delete a stack base branch until every child has been rebased/retargeted.

See `references/decepticon-pr-queue-notes.md` for concrete GitHub CLI and generated-graph observations.
