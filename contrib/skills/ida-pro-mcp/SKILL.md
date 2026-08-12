---
name: ida-pro-mcp
description: "Use when driving IDA Pro via MCP (headless or GUI) for static analysis, annotations, and validation."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: re
  upstream: C:\Users\Admin\.agents\skills\ida-pro-mcp\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\ida-pro-mcp\SKILL.md

# IDA Pro MCP

## Overview
Use `mrexodia/ida-pro-mcp` for one unified GUI/headless workflow. `idalib-mcp` supervises persistent per-database IDALib workers and can adopt or launch GUI IDA instances. Every database-scoped call must pass the session ID returned by `idb_open` or listed by `idb_list`.

No wrapper literally predefines every IDA SDK operation. The installed full profile exposes the server's complete declared tool set plus `py_eval`/`py_exec_file` as an IDAPython escape hatch. Treat arbitrary Python as unsafe and use it only after explicit user direction when no typed tool covers the operation.

## Local Host Setup
Observed Windows installation:
- IDA Professional 9.4: `C:\Program Files\IDA Professional 9.4`
- GUI plugin: `%APPDATA%\Hex-Rays\IDA Pro\plugins\ida_mcp.py` + `ida_mcp\` → `http://127.0.0.1:13337/mcp` **only while IDA is open**
- Pi MCP (`~/.agents/mcp.json`): `ida-pro` = that URL, **`disabled: true` by default**
- Headless supervisor `idalib-mcp` exists but is **not** wired into Pi (avoids always-on workers / timeouts)

### Pi: enable only when IDA is loaded
1. Open the target in IDA Pro (plugin autostarts loopback MCP on `:13337`).
2. `/mcp enable ida-pro` then `/reload` if tools do not appear.
3. Work via the lazy `mcp` proxy against `ida-pro`.
4. `/mcp disable ida-pro` when IDA is closed or the session no longer needs it.

Do not launch `idalib-mcp` for normal interactive RE. The GUI endpoint is loopback-only and disappears when IDA exits.

## When to Use
Use for licensed IDA Pro databases or binaries the user is authorized to analyze. Prefer typed MCP tools. Use direct IDAPython only for a verified gap. Do not use mutation, debugger writes, or target execution without explicit scope.

## Startup and Session Modes
1. Call `idb_list` before opening anything. This prevents duplicate GUI/worker sessions.
2. For headless automation, call `idb_open(input_path, mode=...)` with one mode:
   - `prefer_headless`: adopt matching worker or create headless; default automation path.
   - `force_headless`: never adopt GUI; best for deterministic CI/batch analysis.
   - `prefer_gui` / `force_gui`: supported upstream, but `force_gui` launch registration timed out on this Windows host during validation. Use the observed direct GUI route below until that upstream path is corrected.
3. For GUI work, open the target normally in IDA. The installed plugin autostarts `127.0.0.1:13337`; use the `ida-pro-gui` Hermes server. Verify the process/path and call `server_health` before analysis.
4. For headless sessions, record `session_id`, backend (`worker`), input path, IDB path, image base, architecture, and hashes.
5. Require `auto_analysis_ready`; require `hexrays_ready` before decompilation.

Done when the intended input path maps to exactly one known database and health is ready.

## Analysis Loop
1. `survey_binary` for bounded metadata, segments, entry points, imports, strings, and candidate functions.
2. Use `lookup_funcs`, `func_query`, `imports_query`, `find_regex`, `find_bytes`, and `xref_query` to narrow anchors.
3. Use `analyze_function`, `disasm`, `decompile`, `basic_blocks`, `callees`, and `callgraph` for local semantics.
4. Correct types/frames before renaming. Prefer `rename` dry-run where available.
5. Use `make_signature_for_function` or range signatures and verify uniqueness in the relevant executable segments.
6. Re-query after every annotation or type change; a successful RPC only proves the tool accepted the request.

Paginate large results. Keep addresses as `0x...` strings. Never ask the model to mentally convert precision-sensitive values; use `int_convert`.

## GUI and Headless Rules
- **Headless:** input directories must be writable because IDA may create `.i64` beside the binary. Use an analysis copy. Workers persist across supervisor reconnections and expire by idle TTL.
- **GUI:** use `idb_list` to discover the GUI. A force-launched GUI is user-visible and only the user/UI should close it cleanly. Cursor/selection concepts are GUI-only.
- Do not assume a session filename is accepted as `database`; use the session ID.
- Do not reuse stale session IDs after worker exit or file replacement.

## Mutation Boundary
Low-risk IDB edits include comments, names, types, bookmarks, and function definitions. Higher-risk actions include byte/assembly patches, executable export, debugger writes, and arbitrary Python. Obtain explicit user direction before any mutation, even though the tools are installed and enabled.

For approved edits:
1. Save/snapshot or copy the IDB first.
2. Capture old value/bytes.
3. Apply the smallest typed operation.
4. Read back from IDA and compare decompilation/disassembly.
5. Save only after verification.

Never let `py_eval` import arbitrary sample-controlled modules or access unrelated host files.

## Maintenance
Verify the installed source and connection:
```bash
uv tool list
hermes mcp list
hermes mcp test ida-pro
```

Re-activate IDALib after moving/upgrading IDA:
```bash
uv run "C:\Program Files\IDA Professional 9.4\idalib\python\py-activate-idalib.py"
```

After changing MCP configuration, start a new Hermes session; tool discovery is session-cached.

## Common Pitfalls
1. Confusing GUI plugin mode with the recommended IDALib supervisor. The upstream GUI plugin is deprecated; use `idalib-mcp` adoption/launch.
2. Opening the same binary twice with different paths/casing and then editing the wrong session.
3. Calling decompile before auto-analysis/Hex-Rays readiness.
4. Treating all 68 enabled tools as authorization to mutate.
5. Using arbitrary Python where a typed, auditable tool exists.
6. Saving over the only copy of an IDB or original executable.

## Verification Checklist
- [ ] exact input, hash, IDA version, architecture, image base recorded
- [ ] `idb_list` checked before `idb_open`
- [ ] session ID and backend recorded
- [ ] `server_health` reports analysis ready
- [ ] typed tools used before arbitrary Python
- [ ] mutations explicitly authorized, reversible, and read back
- [ ] GUI opened/closed cleanly when tested
- [ ] exact MCP command and observed results reported

