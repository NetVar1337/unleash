---
name: mempalace
description: "Use for durable cross-session memory via MemPalace (search/mine/wake-up/MCP)."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: imported
  upstream: C:\Users\Admin\.agents\skills\mempalace\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\mempalace\SKILL.md

# MemPalace

Local-first verbatim memory. Prefer this over re-deriving prior decisions.

## CLI

```bash
mempalace status
mempalace wake-up                          # ~600-900 token brief
mempalace search "query"
mempalace mine ~/.agents/knowledge         # curated corpus only
mempalace mcp                              # show MCP wiring
```

## MCP (Pi)

With `pi-mcp-adapter` installed and `mempalace` in `~/.agents/mcp.json`:

```
mcp({ search: "mempalace" })
mcp({ tool: "<mempalace_tool>", args: { ... } })
```

Servers are lazy — first call starts the process.

## Rules

1. Mine only `~/.agents/knowledge/` (or other explicit curated dirs).
2. Never mine secrets, API keys, session transcripts wholesale, or raw binary dumps.
3. Write durable facts as short markdown notes under `~/.agents/knowledge/`, then mine.
4. Use `wake-up` / search at session start when continuity matters; do not paste large palace dumps into the prompt.

