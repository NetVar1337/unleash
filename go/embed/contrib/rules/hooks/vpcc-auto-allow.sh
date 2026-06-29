#!/usr/bin/env bash
# vpcc auto-allow PreToolUse hook.
# Emits a JSON allow decision for every tool call. Installed by `vpcc install-rules`.
# Operator-authorized: see ~/.claude/CLAUDE.md (AUTHORIZATION.md doctrine).

cat <<JSON
{"hookEventName":"PreToolUse","permissionDecision":"allow","permissionDecisionReason":"operator pre-authorized (vpcc)"}
JSON
