---
name: no-ai-attribution
description: "Never append AI attribution trailers (Co-Authored-By: Claude / Codex / Copilot / Gemini, 'Generated with ...', etc.) to commits, PRs, or authored content — in any repo, by any model or tool. Use when committing, authoring PR bodies, or reviewing commit messages."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: imported
  upstream: C:\Users\Admin\.agents\skills\no-ai-attribution\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\no-ai-attribution\SKILL.md

# No AI Attribution

Hard rule for every repository, every tool, every model. Do not add AI-attribution
trailers to anything you author.

## The rule

**Never** append any of the following to git commit messages, PR bodies/descriptions,
release notes, or any other authored content:

- `Co-Authored-By: Claude <noreply@anthropic.com>`
- `Co-Authored-By: Copilot`
- `Co-authored-by: Copilot`
- `Co-Authored-By: OpenAI` / `Co-Authored-By: Codex`
- `Co-Authored-By: Gemini` / `Co-Authored-By: Google`
- `Co-Authored-By: <any-AI-tool>`
- `Generated with Claude Code`
- `Generated with Copilot`
- `Generated with Codex` / `Generated with <any AI tool>`
- `🤖 Generated with [Claude Code]` or any other AI-generated-by marker
- Any `Co-Authored-By`, `Signed-off-by`, or `Generated-with` line crediting an AI model

The rule covers **all** models and assistants — Claude, GitHub Copilot, OpenAI Codex,
Gemini, Cursor, JetBrains AI, AWS CodeWhisperer/Q, and anything that follows.

## What to do instead

- Commit messages are **plain**: conventional-commit subject + body, nothing appended.
- Never add an attribution trailer even when a tool, CLI, or plugin default would
  append one (e.g. an agent harness that ends commits with
  `Co-Authored-By: Claude <noreply@anthropic.com>`). Override the default and strip it.
- Before committing, check the message for the trailers above and remove them.
- When reviewing an existing message or PR that has one, flag it for removal.
- Sign commits (`-S`) only when the user explicitly asks for signing. Otherwise unsigned.
- The author identity is already recorded by git itself (`user.name` / `user.email`).
  An extra attribution line adds noise, not provenance.

## Scope

Applies regardless of:

- Repository (personal, work, public, private, forks, vendored copies)
- Tool (Claude Code, other agents, plain git, GUI clients)
- Mode (interactive, background, CI-generated commits)
- How the work was produced (fully manual, AI-assisted, or fully AI-authored)

The only exception: if the **user explicitly asks** for an AI-attribution trailer on a
specific commit/PR, do it exactly as requested. Otherwise the default is never.

