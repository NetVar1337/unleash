---
name: connect
description: "Use when starting IDA analysis sessions or routing to other IDA skills (CLI/HTTP/MCP setup)."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: imported
  upstream: C:\Users\Admin\.agents\skills\connect\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\connect\SKILL.md

## CLI Help (verbatim `idasql --help`)

```
idasql vX.Y.Z - SQL interface to IDA databases

Usage: idasql -s <file> [-q <query>] [-f <file>] [-i] [--export <file>]

Options:
  -s <file>            IDA database (.idb/.i64) OR raw binary (.exe/.dll/firmware/etc.)
                       — raw binaries trigger fresh idalib analysis and string-list rebuild
                       — legacy 32-bit .idb files upgrade to .i64 and require an explicit reopen
  --token <token>      Auth token for HTTP/MCP server mode (if server requires it)
  -q <sql>             Execute SQL query or semicolon-separated script
  -f <file>            Execute SQL from file
  -i                   Interactive REPL mode
  -w, --write          Save database on exit (persist changes)
  --export <file>      Export tables to SQL file (local mode only)
  --export-tables=X    Tables to export: * (all, default) or table1,table2,...
  --http [port]        Start HTTP REST server (default: 8080, local mode only)
  --bind <addr>        Bind address for HTTP/MCP server (default: 127.0.0.1)
  --mcp [port]         Start MCP server (default: random port, use in -i mode)
                       Or use .mcp start in interactive mode
  -h, --help           Show this help
  --version            Show version

Examples:
  idasql -s test.i64 -q "SELECT name, size FROM funcs LIMIT 10"
  idasql -s test.i64 -f queries.sql
  idasql -s test.i64 -i
  idasql -s test.i64 --export dump.sql
  idasql -s test.i64 --http 8080
  idasql -s sample.exe --http            # raw PE: idalib auto-analyzes, then serves SQL (default port 8080)
  idasql -s firmware.bin -q "SELECT * FROM binary"
  idasql -s test.i64 --mcp 9000
```

**Key takeaway:** `-s` accepts either an existing IDA database **or** a raw binary. No
manual `idat -A -B` / `ida -B` pre-step is needed — point `-s` straight at the
`.exe`/`.dll`/firmware/etc. and idalib creates the database on first open.

---

## Additional Resources

- Canonical schema catalog: [references/schema-catalog.md](references/schema-catalog.md)
- CLI reference, REPL commands, server modes, runtime controls: [references/cli-reference.md](references/cli-reference.md)
  (includes `.pin` autostart: the IDA plugin can auto-start a pinned HTTP/MCP server when a database is opened)

### Autostart pinning (HTTP/MCP)

`.pin http|mcp [bindinterface] [port]` pins a server so the **IDA plugin
auto-starts it when the database opens** (the pin lives in the IDB; CLI changes
persist only with `-w`). The port is **optional**:

- **Fixed port** — `.pin http 0.0.0.0 8080` → autostarts on that stable port every
  launch (use for stable automation).
- **Random port** — `.pin http` (port omitted, stored as `0`) → autostarts on a
  **fresh random port each launch** (a real pinned setting, not "unset"); read the
  chosen port from the live status. `.pin status` shows `<random port each launch>`.

`[bindinterface]` is the host/interface (default `127.0.0.1`). `.pin on|off http|mcp`
toggles autostart without dropping the host/port; `.pin clear [http|mcp|all]` removes
it. `.http start` / `.mcp start` with no port reuse the pinned host/port. Full grammar:
[references/cli-reference.md](references/cli-reference.md).
- Optimization quality gate: [references/optimization-checklist.md](references/optimization-checklist.md)
- HTTP server guide: [references/server-guide.md](references/server-guide.md)

---

## Quick Start Guardrails

- Always provide `-s <file>`. The file can be either an IDA database
  (`.idb` / `.i64`) **or** a raw binary (`.exe`, `.dll`, firmware blob,
  etc.) — raw binaries trigger fresh idalib auto-analysis and string-list
  rebuild. Do **not** pre-create an IDB with `idat -A -B`;
  `idasql -s raw.exe` does it in one shot.
- If `idasql -s legacy.idb ...` exits with code `3` and stdout JSON containing
  `{"status":"upgraded","reopen_with":"..."}`, repeat the same requested
  operation with `-s <reopen_with>`. Server modes exit before binding in this
  case.
- Use `--write` when you want edits persisted on exit, including HTTP/MCP server shutdown.
- SQL inputs to `-q`, HTTP `/query`, MCP `idasql_query`, and plugin CLI
  may be one statement or a semicolon-separated script. Semicolons inside
  quoted strings are safe.
- **All `/query` responses use the canonical script envelope** — single
  statement = array of one:
  `{success, statement_count, results:[{statement_index, success,
  columns, rows, row_count, elapsed_ms, error}], row_count_total,
  elapsed_ms_total, first_error_index}`.
  Fail-fast is the default. Only the REPL/plugin `.http` server honors
  `?continue_on_error=1` in the query string to run every statement
  regardless of earlier failures; the CLI `--http` handler ignores it
  and MCP has no such argument.
- Discover schema before writing queries:
  - REPL: `.schema <table>`
  - SQL: `PRAGMA table_xinfo(<table>);`
- Start orientation with `SELECT * FROM binary;`.

---

## Schema Catalog

Canonical column shapes and owner-skill mapping live in
`references/schema-catalog.md`. Sourced from `pragma_table_list` +
`pragma_table_xinfo`. Use it before assuming column names for
less-common surfaces.

---

## Session Bootstrap Contract

Use this exact startup flow before deep analysis:

1. Connect to database (`-s`, `-i`, or `--http`). `-s` accepts either an existing IDB
   (`.idb`/`.i64`) or a raw binary (`.exe`/`.dll`/firmware/etc.) — never pre-build an
   IDB with `idat`; let `idasql -s raw.exe` do it.
   If a legacy `.idb` upgrades and returns `status:"upgraded"`, restart with the
   JSON `reopen_with` path before running orientation.
2. Run orientation query:
```sql
SELECT * FROM binary;
```
3. Validate key entities exist:
```sql
SELECT COUNT(*) AS funcs FROM funcs;
SELECT COUNT(*) AS xrefs FROM xrefs;
SELECT COUNT(*) AS strings FROM strings;
```
4. Introspect schema for target surfaces before authoring complex SQL:
```sql
PRAGMA table_xinfo(funcs);
PRAGMA table_xinfo(xrefs);
```
5. Route to domain skill using routing matrix below.

---

## Global Agent Contracts

These contracts apply across all idasql skills and should be treated as one shared agent behavior model.

### Read-First Contract
- Read current state first (`SELECT`) before writes (`INSERT`/`UPDATE`/`DELETE`).
- Confirm target precision using stable identifiers (`addr`, `func_addr`, `idx`, `label_num`).

### Anti-Guessing Contract
- Do not assume columns/types for long-tail surfaces.
- Introspect via `.schema` or `PRAGMA table_xinfo(...)` before issuing uncertain queries.

### Mandatory Mutation Loop
1. Read current state.
2. Apply mutation.
3. Refresh if needed (`decompile(..., 1)` for decompiler surfaces).
4. Re-read and verify expected change.

### Performance Contract
- Always constrain high-cost surfaces (`xrefs`, `instructions`, `ctree*`, `pseudocode`) by key columns.
- For decompiler surfaces, enforce `func_addr = X` unless explicitly asked for broad scans.
- For raw dirtree browsing, prefer `tree = ?` plus `path`, `path LIKE`, or `parent_path`; for normal organization use `funcs.folder_path` and `types.folder_path`.
- `dirtree_entries` is read-only diagnostics. Recursive folder delete and raw recovery/link operations are intentionally not SQL surfaces.

### Failure Recovery Contract
- On `no such table/column`: introspect schema and retry.
- On empty results: validate address range, table freshness (`rebuild_strings()`), and runtime capabilities.
- On timeout: narrow scope, add constraints, paginate, or split query.

### Output Contract
- **Selection** - decide *whether and how much* to surface from user intent. Answer questions directly ("biggest is `main`, 500 bytes"); show supporting rows only when they help the user verify; don't dump full tables unprompted; never surface data fetched only as an intermediate reasoning step.
- **Fidelity** - when you *do* present code/data, show the real artifact (decompilation, actual rows), never a paraphrase.
- **Mechanics** - the HTTP `/query` response is a JSON envelope (`{success, results:[{columns,rows,...}]}`). Consume it directly and render in your reply. Do **not** pipe responses through `python`/`jq` to pre-render a table - that discards the `success`/`elapsed_ms`/`error` fields and makes you reason over a lossy view. The CLI (`-q`/`-f`) already prints a table. Reserve `jq`/`python` for extracting a value to feed a later query. (For direct terminal/pipe use the server can emit `?format=text|csv|tsv`; as an agent, consume `json`.)

---

## Skill Routing Matrix (Intent -> Skill)

Use this deterministic mapping for initial routing:

| User intent | Primary skill | Typical first query |
|-------------|---------------|---------------------|
| "what does this binary do?" / triage | `analysis` | `SELECT * FROM entries;` |
| disassembly, segments, instructions | `disassembly` | `SELECT * FROM funcs LIMIT 20;` |
| function/type folders, review buckets, folder lifecycle | `annotations` / `types` | `SELECT addr, name, folder_path FROM funcs WHERE folder_path LIKE 'idasql/%';` |
| xrefs/callers/callees/import dependencies | `xrefs` | `SELECT * FROM xrefs WHERE to_addr = ...;` |
| find functions/types/labels/members by name pattern | `grep` | `SELECT name, kind, addr FROM grep WHERE pattern = 'main' LIMIT 20;` |
| strings/bytes/pattern search | `data` | `SELECT * FROM strings LIMIT 20;` |
| decompile/pseudocode/ctree/lvars | `decompiler` | `SELECT decompile(0x...);` |
| comments/renames/retyping/bookmarks | `annotations` | `SELECT ...` on target row before update |
| type creation/struct/enum/member work | `types` | `SELECT * FROM types LIMIT 20;` |
| breakpoints/patching | `debugger` | `SELECT * FROM breakpoints;` |
| persistent key/value notes | `storage` | `SELECT * FROM netnode_kv LIMIT 20;` |
| SQL function lookup/signature recall | `functions` | `SELECT * FROM pragma_function_list;` |
| enumerate runtime settings / timeouts / queue config | `connect` | `SELECT * FROM runtime_settings;` (read-only; change via `PRAGMA idasql.<key> = <value>`) |
| live IDA UI context questions | `ui-context` | `SELECT get_ui_context_json();` (when available) |
| IDA SDK-only logic not in SQL surfaces | `idapython` | `PRAGMA idasql.enable_idapython = 1; SELECT idapython_snippet('print(...)');` |
| recursive source/structure recovery | `re-source` | start from function + recurse/handoff |

When prompts span domains, execute in this order:
1. Orientation in `connect`
2. Primary domain skill
3. Adjacent skills for enrichment (for example `xrefs` + `decompiler` + `annotations`)

---

## Cross-Skill Execution Recipes

### Recipe: Unknown binary triage -> suspicious function deep dive -> annotate
1. `analysis`: identify candidates from imports/strings/call patterns.
2. `xrefs`/`disassembly`: map call graph and call sites.
3. `decompiler`: inspect logic and variable semantics.
4. `annotations`: apply comments/renames/types with mutation loop.

### Recipe: String IOC -> reference graph -> patch
1. `data`: locate candidate strings and addresses.
2. `xrefs`: map references to caller functions.
3. `debugger` or `annotations`: patch or annotate specific sites.

### Recipe: Type recovery from pseudocode
1. `decompiler`: inspect lvars, call args, and ctree patterns.
2. `types`: create/refine structs/enums and apply declarations.
3. `annotations`: finalize naming/comments and verify rendered pseudocode.

---

## UI Context Routing

For prompts like "what am I looking at?", "what's selected?", "what is on the screen?", "look at what I'm doing", or references to "this/current/that", use the dedicated `ui-context` skill.

`ui-context` owns:
- `get_ui_context_json()` capture/reuse policy
- temporal reference rules (`this` vs `that`)
- response template, examples, and fallback messaging

Runtime caveat:
- `get_ui_context_json()` is plugin GUI runtime only, not idalib/CLI mode.
- If unavailable, state that UI context is unavailable and continue with non-UI SQL workflows.

---

## binary

Database orientation surface for quick session metadata.
This is metadata-only and not a replacement for UI context capture.

Key/value shape — `(key TEXT, value TEXT, type TEXT)`, one row per metadata
fact, `type ∈ {string, hex, bool, int}`. The shape and the canonical key names
are shared across the tool family, so the same `WHERE key = '…'` query works on
every engine. The `summary` row is emitted first, so `SELECT * FROM binary`
shows the digest up top.

| Key | Type | Description |
|-----|------|-------------|
| `summary` | string | One-line database summary (first row) |
| `tool_name` | string | Always `idasql` |
| `tool_version` | string | IDASQL build version (canonical name) |
| `idasql_version` | string | IDASQL build version (legacy-named duplicate) |
| `processor` | string | Processor/module name |
| `filetype` | string | Loader file-type description (e.g. PE, ELF) |
| `image_base` | hex | Image base address |
| `entry_point` | hex | Entry/start address (was the `start_addr` column) |
| `min_addr` | hex | Minimum address in database |
| `max_addr` | hex | Maximum address in database |
| `is_64bit` | bool | `true`/`false` |
| `bits` | int | Application bitness: 16, 32, or 64 |
| `endianness` | string | `little` or `big` |
| `filename` | string | Bare input filename (no path) |
| `entry_name` | string | Entry symbol name (if known) |
| `funcs_count` | int | Number of detected functions |
| `segments_count` | int | Number of segments |
| `names_count` | int | Number of named addresses |
| `strings_count` | int | Current IDA string-list count |
| `input_file_path` | string | Original input file path recorded in the IDB |
| `idb_path` | string | On-disk path of the IDB/I64 (may differ if moved) |
| `md5` | string | Lowercase hex MD5 of the input file (empty if unavailable) |
| `sha256` | string | Lowercase hex SHA-256 of the input file (empty if unavailable) |

Use the `filename`, `idb_path`, `md5`, or `sha256` keys to confirm which
binary/instance this connection is bound to.

```sql
SELECT * FROM binary;
SELECT key, value FROM binary WHERE key IN ('filename','idb_path','md5','sha256');
SELECT value FROM binary WHERE key = 'idasql_version';
```

For canonical schema and owner mapping, see `references/schema-catalog.md`.

---

## What is IDA and Why SQL?

**IDA Pro** is the industry-standard disassembler and reverse engineering tool. It analyzes compiled binaries (executables, DLLs, firmware) and produces:
- **Disassembly** - Human-readable assembly code
- **Functions** - Detected code boundaries with names
- **Cross-references** - Who calls what, who references what data
- **Types** - Structures, enums, function prototypes
- **Decompilation** - C-like pseudocode (with Hex-Rays plugin)

**IDASQL** exposes all this analysis data through SQL virtual tables, enabling:
- Complex queries across multiple data types (JOINs)
- Aggregations and statistics (COUNT, GROUP BY)
- Pattern detection across the entire binary
- Scriptable analysis without writing IDA plugins or IDAPython scripts

---

## Core Concepts for Binary Analysis

### Addresses (ea_t)
Everything in a binary has an **address** - a memory location where code or data lives. IDA uses `ea_t` (effective address) as unsigned 64-bit integers. SQL shows these as integers; use `printf('0x%X', addr)` for hex display.

Address-taking SQL functions accept:
- integer EA values (preferred for deterministic scripts)
- numeric strings (`'4198400'`, `'0x401000'`)
- symbol names resolved with `get_name_ea(BADADDR, name)` (global names)

Most table predicates should compare address columns to integer EAs such as
`addr = 0x401000`. `applied_types.addr` is an intentional exception for
equality filters/writes: it also accepts numeric strings and symbol names so
type application can target names directly.

Examples:
```sql
SELECT decompile('DriverEntry');
UPDATE applied_types
SET decl = 'NTSTATUS DriverEntry(PDRIVER_OBJECT, PUNICODE_STRING);'
WHERE addr = 'DriverEntry';
SELECT (SELECT comment FROM comments WHERE addr = 0x401000 LIMIT 1);
```

Read address comments from the `comments` table after resolving the target EA.

If a symbol cannot be resolved, SQL functions return an explicit error like:
`Could not resolve name to address: <name>`.

Local label lookup that depends on a specific `from` context is not consulted by default (`BADADDR` resolution). Use explicit numeric EAs when needed.

### Functions
IDA groups code into **functions** with:
- `addr` / `start_addr` - Where the function begins
- `end_addr` - Where it ends
- `name` - Assigned or auto-generated name (e.g., `main`, `sub_401000`)
- `size` - Total bytes in the function

There will be addresses and disassembly listing not belonging to a function. IDASQL can still get the bytes, disassembly listing ranges, etc.
For single-EA disassembly (code or data), prefer `disasm_at(addr[, context])` over function-scoped queries.

### Cross-References (xrefs)
Binary analysis is about understanding **relationships**:
- **Code xrefs** - Function calls, jumps between code
- **Data xrefs** - Code reading/writing data locations, or data referring to other data (pointers)
- `from_addr` -> `to_addr` represents "address X references address Y"
Use table: `xrefs(from_addr, to_addr, type, is_code)`.

### Segments

Use table: `segments(start_addr, end_addr, name, class, perm)` (full CRUD).
`UPDATE start_addr` rebases a segment, `end_addr` resizes it.

Memory is divided into **segments** with different purposes. For example, a typical PE file, has these segments:

- `.text` - Executable code (typically)
- `.data` - Initialized global data
- `.rdata` - Read-only data (strings, constants)
- `.bss` - Uninitialized data

Of course, segment names and types can vary. You may query the `segments` table to understand memory layout.

### Basic Blocks
Within a function, **basic blocks** are straight-line code sequences:
- No branches in the middle
- Single entry, single exit
- Useful for control flow analysis
Use table: `blocks(start_addr, end_addr, func_addr, size)`.

### Decompilation (Hex-Rays)
The **Hex-Rays decompiler** converts assembly to C-like **pseudocode**:
- **ctree** - The Abstract Syntax Tree of decompiled code
- **lvars** - Local variables detected by the decompiler
- Much easier to analyze than raw assembly

Core decompiler surfaces:
- `decompile(addr)` (**PRIMARY read/display surface**)
  - Returns the entire function as one text block.
  - Each output line is prefixed for address grounding:
    - Addressed line: `/* 401010 */ ...`
    - Non-anchored line: `/*          */ ...` (no address anchor for that line)
  - Use this first when the user asks to "decompile", "show code", "show pseudocode", or "explain function logic".
- `pseudocode` table (**structured/edit surface**)
  - Use for line-level filtering (`func_addr`, `addr`, `line_num`) and comment writes keyed by `addr + comment_placement`.
  - Resolve a writable pseudocode anchor first; do not assume `addr == func_addr`.
  - Not the preferred display surface for full-function code.
- `ctree` and `ctree_call_args` for AST-level analysis
- `ctree_lvars` for local variable rename/type/comment updates

---

## Performance Rules

### CRITICAL: Constraint Pushdown

Some tables have **optimized filters** that use efficient IDA SDK APIs:

| Table | Optimized Filter | Without Filter |
|-------|------------------|----------------|
| `instructions` | `func_addr = X` | O(all instructions) - SLOW |
| `blocks` | `func_addr = X` | O(all blocks) |
| `xrefs` | `to_addr = X` or `from_addr = X` | O(all xrefs) |
| `pseudocode` | `func_addr = X` | **Decompiles ALL functions** |
| `ctree*` | `func_addr = X` | **Decompiles ALL functions** |

**Always filter decompiler tables by `func_addr`!**

### Use Integer Comparisons

```sql
-- SLOW: String comparison
WHERE mnemonic = 'call'

-- FAST: Integer comparison
WHERE itype IN (16, 18)  -- x86 call opcodes
```

### O(1) Random Access

```sql
-- SLOW: O(n) - sorts all rows
SELECT addr FROM funcs ORDER BY RANDOM() LIMIT 1;

-- FAST: O(1) - direct index access
SELECT addr
FROM funcs
WHERE rowid = ABS(RANDOM()) % (SELECT COUNT(*) FROM funcs);
```

### CTE-First Mutation Workflow

For instruction lifecycle edits, use a CTE to identify precise targets first, then mutate:

```sql
WITH target AS (
    SELECT addr
    FROM instructions
    WHERE func_addr = 0x401000
    ORDER BY addr DESC
    LIMIT 1
)
DELETE FROM instructions
WHERE addr IN (SELECT addr FROM target);

SELECT make_code_range(addr, end_addr) FROM funcs WHERE addr = 0x401000;
```

This keeps mutation scope explicit and predictable for both humans and agents.

---

## Summary: When to Use What

| Goal | Table/Function |
|------|----------------|
| List all functions | `funcs` (cols: `addr`, `name`, `end_addr`, `prototype`, `flags`, …) → `disassembly` |
| Functions by return type | `funcs WHERE return_is_integral = 1` |
| Functions by arg count | `funcs WHERE arg_count >= N` |
| Void functions | `funcs WHERE return_is_void = 1` |
| Pointer-returning functions | `funcs WHERE return_is_ptr = 1` |
| Functions by calling convention | `funcs WHERE calling_conv = 'fastcall'` |
| Find who calls what | `xrefs` with `is_code = 1` |
| Find data references | `xrefs` with `is_code = 0` |
| Analyze imports | `imports` (cols: `addr`, `name`, `ordinal`, `module`) → `xrefs` / `analysis` |
| Find strings | `strings` (cols: `addr`, `length`, `content`) → `data` |
| Configure string types | `rebuild_strings(minlen, types)` |
| Instruction analysis | `instructions WHERE func_addr = X` |
| Recreate deleted instructions | `make_code(addr)`, `make_code_range(start, end)` |
| Apply/clear address type declarations | `applied_types` (`INSERT`, `UPDATE decl`, `DELETE`) |
| Apply/clear call-site prototypes | `UPDATE disasm_calls SET callee_type = ... WHERE addr = X` |
| Create function at EA | `INSERT INTO funcs(addr) VALUES (...)` |
| View function disassembly | `disasm_func(addr)` or `disasm_range(start, end)` |
| View decompiled code | `decompile(addr)` |
| UI/screen context questions | `ui-context` skill (`get_ui_context_json()`, plugin UI only) |
| Edit decompiler comments | `Resolve writable anchor, then UPDATE pseudocode SET comment = '...' WHERE func_addr = X AND addr = Y` |
| AST pattern matching | `ctree WHERE func_addr = X` |
| Call patterns | `ctree_v_calls`, `disasm_calls` |
| Control flow | `ctree_v_loops`, `ctree_v_ifs` |
| Return value analysis | `ctree_v_returns` |
| Functions returning specific values | `ctree_v_returns WHERE return_num = 0` |
| Pass-through functions | `ctree_v_returns WHERE returns_arg = 1` |
| Wrapper functions | `ctree_v_returns WHERE returns_call_result = 1` |
| Variable analysis | `ctree_lvars WHERE func_addr = X` |
| Type information | `types`, `types_members` |
| Function signatures | `types_func_args` (with type classification) |
| Functions by return type | `types_func_args WHERE arg_index = -1` |
| Typedef-aware type queries | `types_func_args` (surface vs resolved) |
| Hidden pointer types | `types_func_args WHERE is_ptr = 0 AND is_ptr_resolved = 1` |
| Manage breakpoints | `breakpoints` (full CRUD) |
| Modify segments | `segments` (INSERT/UPDATE/DELETE) |
| Rename decompiler labels | `UPDATE ctree_labels SET name=... WHERE func_addr=... AND label_num=...` |
| Delete instructions | `instructions` (DELETE converts to unexplored bytes) |
| Recreate instructions | `make_code`, `make_code_range` |
| Bulk patch from file bytes | `load_file_bytes(path, file_offset, addr, size[, patchable])` |
| EA to physical offset mapping | `bytes.fpos` on mapped byte rows (`NULL` means no file offset) |
| Create types | `types` (INSERT struct/union/enum) |
| Add struct members | `types_members` (INSERT) |
| Add enum values | `types_enum_values` (INSERT) |
| Modify database | `funcs`, `names`, `comments`, `bookmarks` (INSERT/UPDATE/DELETE) |
| Store custom key-value data | `netnode_kv` (full CRUD, persists in IDB) |
| Entity search (structured) | `grep` skill + `grep WHERE pattern = '...'` |

**Remember:** Always use `func_addr = X` constraints on instruction and decompiler tables for acceptable performance.

---

## Error Handling

- **No Hex-Rays license:** Decompiler tables (`pseudocode`, `ctree*`, `ctree_lvars`) will be empty or unavailable
- **No constraint on decompiler tables:** Query will be extremely slow (decompiles all functions)
- **Invalid address:** Containing-function table lookups return no row; use a scalar subquery when you need a nullable scalar result
- **Missing function:** JOINs may return fewer rows than expected

