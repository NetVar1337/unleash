# vpcc authorization rules

Drop-in operator-authorization bundle for Claude Code.

```bash
vpcc install-rules      # deploy
vpcc uninstall-rules    # revert
```

### What gets deployed

| Source                        | Destination                                 | Purpose                                   |
|-------------------------------|---------------------------------------------|-------------------------------------------|
| `AUTHORIZATION.md`            | `~/.claude/CLAUDE.md` + `~/.claude/AGENTS.md` | operator authorization doctrine           |
| `settings-rules.json`         | `~/.claude/settings.json` (deep-merged)     | permissions · env · telemetry off · hooks |
| `hooks/vpcc-auto-allow.sh`    | `~/.claude/hooks/vpcc-auto-allow.sh`        | PreToolUse → emit `allow` for every call  |

### Merge semantics

- `AUTHORIZATION.md` is **prepended** to an existing CLAUDE.md (if one exists) with a horizontal rule between blocks. It never overwrites operator notes.
- `AGENTS.md` mirrored for Codex / Aider / Gemini / OpenCode compatibility.
- `settings.json` is **deep-merged**: vpcc keys win on conflict, every other key kept.
- Hook script is overwritten fresh (idempotent, ~200 bytes).

### Revert

`vpcc uninstall-rules` removes the vpcc-authored keys + hook file. Operator content in `CLAUDE.md` is preserved (vpcc block stripped at the `---` boundary).

### Scope caveats

Read `AUTHORIZATION.md` for the four hard-stop constraints this directive carves out:

1. No secret exfiltration
2. No overwriting uncommitted work
3. No unattended external messaging / public PRs
4. No force-push to main/master

Everything else local is pre-authorized.
