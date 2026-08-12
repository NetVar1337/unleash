---
name: functions
description: "Use as the idasql SQL function reference catalog (signatures, params, examples)."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: imported
  upstream: C:\Users\Admin\.agents\skills\functions\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\functions\SKILL.md

This skill is a **comprehensive catalog** of every idasql SQL function. Use it to look up any function signature, parameters, and usage.

---

## Disassembly

| Function | Description |
|----------|-------------|
| `disasm_at(addr)` | Canonical listing line for containing head (works for code/data) |
| `disasm_at(addr, n)` | Canonical listing line with +/- `n` neighboring heads |
| `disasm(addr)` | Single disassembly line at address |
| `disasm(addr, n)` | Next N instructions from address (count-based, not boundary-aware) |
| `disasm_range(start, end)` | All disassembly lines in address range [start, end) |
| `disasm_func(addr)` | Full disassembly of function containing address |
| `make_code(addr)` | Create instruction at address (returns 1 if already code or created) |
| `make_code_range(start, end)` | Create instructions in [start, end), returns number created |

```sql
SELECT disasm_at(0x401000);
SELECT disasm_at(0x401000, 2);
SELECT disasm_func(addr) FROM funcs WHERE name = '_main';
SELECT disasm_range(0x401000, 0x401100);
SELECT disasm(0x401000);
SELECT disasm(0x401000, 5);
SELECT make_code(0x401000);
SELECT make_code_range(0x401000, 0x401100);
```

Function creation is table-driven (not a SQL function):
```sql
INSERT INTO funcs (addr) VALUES (0x401000);
```

---

## Byte Access and Patching

Byte reads, bulk reads, and patching all go through the `bytes` table.
`load_file_bytes(...)` writes a host file's bytes into the IDB at a given
range; it returns `1` on success, `0` on failure.

| Function | Description |
|----------|-------------|
| `load_file_bytes(path, file_offset, addr, size[, patchable])` | Write host file bytes into IDB memory at the target range |
| `blob_concat(value)` | built-in aggregate — concatenate byte values into one BLOB |

```sql
-- Read 16 bytes as uppercase hex
SELECT hex(blob_concat(value))
FROM (SELECT value FROM bytes WHERE start_addr = 0x401000 AND n = 16 ORDER BY addr);

-- Read 64 bytes as a BLOB (its byte length; bind the raw BLOB itself, or wrap in
-- hex() for a text-safe transport)
SELECT length(blob_concat(value)) AS blob_len
FROM (SELECT value FROM bytes WHERE start_addr = 0x401000 AND n = 64 ORDER BY addr);

-- Patch 1 byte and 4 LE bytes
UPDATE bytes SET value = 0x90 WHERE addr = 0x401000;
UPDATE bytes SET dword = 0x90909090 WHERE addr = 0x401000;
SELECT value AS current, original_value AS original FROM bytes WHERE addr = 0x401000;
DELETE FROM bytes WHERE addr = 0x401000;                   -- revert
```

For composable row-shaped reads use the `bytes` table directly:
`SELECT addr, value FROM bytes WHERE addr >= :start AND addr < :end ORDER BY addr`.
Use `heads` for item size/type metadata.

---

## Binary Search

Use the `byte_search` table for raw bytes/opcodes. It is table-shaped so results can be filtered, joined, grouped, and limited directly.

| Column | Description |
|--------|-------------|
| `addr` | Match address |
| `matched_hex` | Matched bytes rendered as hex text |
| `matched_bytes` | Matched bytes as a BLOB |
| `size` | Match size in bytes |
| `pattern` | Hidden required input: IDA byte pattern |
| `start_addr` | Hidden optional inclusive lower bound |
| `end_addr` | Hidden optional exclusive upper bound |
| `max_results` | Hidden optional generator cap |

**Pattern syntax (IDA native):**
- `"48 8B 05"` - Exact bytes (hex, space-separated)
- `"48 ? 05"` or `"48 ?? 05"` - `?` = any byte wildcard (whole byte only)
- `"(01 02 03)"` - Alternatives (match any of these bytes)

```sql
SELECT addr, matched_hex, size
FROM byte_search
WHERE pattern = '48 8B ? 00'
LIMIT 10;

SELECT printf('0x%llX', addr) AS addr
FROM byte_search
WHERE pattern = 'CC CC CC'
ORDER BY addr
LIMIT 1;
```

**Optimization Pattern:**
```sql
-- Count unique functions containing RDTSC (opcode: 0F 31)
SELECT COUNT(DISTINCT f.addr) as count
FROM byte_search b
JOIN funcs f ON b.addr >= f.addr AND b.addr < f.end_addr
WHERE b.pattern = '0F 31';
```

---

## Names & Functions

Use table lookups for address and containing-function metadata. Resolve symbol names to integer EAs before using these patterns.

| Pattern | Description |
|---------|-------------|
| `SELECT name FROM names WHERE addr = :addr LIMIT 1` | Name at address |
| `SELECT name FROM funcs WHERE :addr >= addr AND :addr < end_addr LIMIT 1` | Function containing address |
| `SELECT addr FROM funcs WHERE :addr >= addr AND :addr < end_addr LIMIT 1` | Start of containing function |
| `SELECT end_addr FROM funcs WHERE :addr >= addr AND :addr < end_addr LIMIT 1` | End of containing function |

Function count and index lookup are table-driven:

```sql
SELECT COUNT(*) AS function_count FROM funcs;
SELECT addr FROM funcs WHERE rowid = 0;
```

---

## Cross-References

Cross-reference edge queries are table-driven:

```sql
SELECT from_addr, to_addr, type, is_code, from_func
FROM xrefs
WHERE to_addr = 0x401000;

SELECT from_addr, to_addr, type, is_code, from_func
FROM xrefs
WHERE from_addr = 0x401000;

SELECT from_addr, to_addr, type, is_code, from_func
FROM xrefs
WHERE from_func = 0x401000;
```

---

## Navigation

Use `heads` ordering for defined-item navigation and SQLite formatting functions for display strings. Address equality/range filters are optimized; `ORDER BY addr` or `ORDER BY addr DESC` is consumed for next/previous-item lookups.

```sql
SELECT addr
FROM heads
WHERE addr > 0x401000
ORDER BY addr
LIMIT 1;

SELECT addr
FROM heads
WHERE addr < 0x401000
ORDER BY addr DESC
LIMIT 1;

SELECT printf('0x%llx', addr) AS address_hex
FROM heads
LIMIT 10;
```

Segment lookup is table-driven:

```sql
SELECT name
FROM segments
WHERE 0x401000 >= start_addr
  AND 0x401000 < end_addr
LIMIT 1;
```

---

## Comments

Read comments through the `comments` table:

```sql
SELECT COALESCE(NULLIF(comment, ''), NULLIF(rpt_comment, '')) AS comment
FROM comments
WHERE addr = 0x401000
LIMIT 1;
```

Write comments through the table:

```sql
INSERT INTO comments(addr, comment) VALUES (0x401000, 'regular comment');
INSERT INTO comments(addr, rpt_comment) VALUES (0x401000, 'repeatable comment');
-- Replace an existing comment in place
UPDATE comments SET comment = 'revised comment' WHERE addr = 0x401000;
-- Remove a comment
DELETE FROM comments WHERE addr = 0x401000;
```

Note: `INSERT` at an address that already has a comment **replaces** it (one comment per slot per EA). `UPDATE` is equivalent.

---

## Modification

| Surface | Description |
|---------|-------------|
| `applied_types(addr, decl, ordinal, type_name)` | Read/apply/replace/clear C declarations at addresses |
| `parse_decls(text)` | Import C declarations (struct/union/enum/typedef) into local types |

Preferred SQL write surface for function metadata:
- `UPDATE funcs SET name = '...', prototype = '...', folder_path = 'idasql/review/annotated' WHERE addr = ...`
- `INSERT INTO names(addr, name) VALUES (..., '...')` or `UPDATE names SET name = '...' WHERE addr = ...`
- `prototype` maps to `applied_types` behavior and invalidates decompiler cache.
- `folder_path` moves functions in IDA's Function window tree; create/rename/delete empty folders through `dirtree_folders`.
- For per-call indirect-call typing, update `disasm_calls.callee_type`.

---

## Python Execution

| Function | Description |
|----------|-------------|
| `idapython_snippet(code[, sandbox])` | Execute Python snippet and return captured output text |
| `idapython_file(path[, sandbox])` | Execute Python file and return captured output text |

Runtime guard:
```sql
PRAGMA idasql.enable_idapython = 1;
```

```sql
SELECT idapython_snippet('print("hello from idapython")');
SELECT idapython_file('C:/temp/script.py');
SELECT idapython_snippet('counter = globals().get("counter", 0) + 1; print(counter)', 'alpha');
```

---

## Context Awareness (Plugin UI)

| Function | Description |
|----------|-------------|
| `get_ui_context_json()` | Return current UI/widget/context JSON for context-aware prompts (registered everywhere; live UI in the GUI plugin, a `source:"cli"` stub under idalib/CLI) |

```sql
SELECT get_ui_context_json();
```

---

## Item Analysis

Use `heads` for item classification, size, and raw flags:

```sql
SELECT addr, size, type, flags, disasm
FROM heads
WHERE addr = 0x401000;
```

---

## Instruction Details

Use `instructions` and `instruction_operands` for decoded instruction facts. `instruction_operands` exposes one row per non-void operand.

```sql
SELECT addr, itype, mnemonic
FROM instructions
WHERE func_addr = 0x401000
LIMIT 10;

SELECT opnum, text, type_code, type_name, value
FROM instruction_operands
WHERE addr = 0x401000
ORDER BY opnum;

SELECT i.addr, i.itype, i.mnemonic, i.size, o.opnum, o.text, o.type_name, o.value
FROM instructions i
LEFT JOIN instruction_operands o
  ON o.addr = i.addr AND o.addr = 0x401000
WHERE i.addr = 0x401000
ORDER BY o.opnum;
```

---

## Decompilation

> **Hex-Rays required.** Every function in this section (`decompile`, `call_arg_addrs`,
> the `set_union_selection*` / `get_union_selection*` families, `call_arg_item`,
> `ctree_item_at`, and the `set_numform*` / `get_numform*` families) is only registered
> when the Hex-Rays decompiler is available. On a build/runtime without Hex-Rays they are
> absent and calling one returns `no such function`. Probe with `SELECT decompile(<addr>)`
> before relying on them.

| Function | Description |
|----------|-------------|
| `decompile(addr)` | **PREFERRED** — Full pseudocode with line prefixes |
| `decompile(addr, 1)` | Force re-decompilation (use after writes/renames) |
| `call_arg_addrs(call_addr)` | JSON array of persisted argument-loader instruction EAs |
| `set_union_selection(func_addr, addr, path)` | Set/clear union selection path at EA |
| `set_union_selection_item(func_addr, item_id, path)` | Set/clear union selection path by `ctree.item_id` |
| `set_union_selection_addr_arg(func_addr, addr, arg_idx, path[, callee])` | **PREFERRED** call-arg targeting helper |
| `call_arg_item(func_addr, addr, arg_idx[, callee])` | Resolve call-arg coordinate to explicit `arg_item_id` |
| `ctree_item_at(func_addr, addr[, op_name[, nth]])` | Resolve generic expression coordinate to `ctree.item_id` |
| `set_union_selection_addr_expr(func_addr, addr, path[, op_name[, nth]])` | Set/clear union selection via expression coordinate |
| `get_union_selection(func_addr, addr)` | Read union selection path JSON at EA |
| `get_union_selection_item(func_addr, item_id)` | Read union selection path JSON by `ctree.item_id` |
| `get_union_selection_addr_arg(func_addr, addr, arg_idx[, callee])` | Read union selection JSON via call-arg coordinate |
| `get_union_selection_addr_expr(func_addr, addr[, op_name[, nth]])` | Read union selection JSON via expression coordinate |
| `set_numform(func_addr, addr, opnum, spec)` | Set/clear numform by EA + operand index |
| `get_numform(func_addr, addr, opnum)` | Read numform JSON by EA + operand index |
| `set_numform_item(func_addr, item_id, opnum, spec)` | Set/clear numform by ctree item id |
| `get_numform_item(func_addr, item_id, opnum)` | Read numform JSON by ctree item id |
| `set_numform_addr_arg(func_addr, addr, arg_idx, opnum, spec[, callee])` | Set/clear numform via call-arg coordinate |
| `get_numform_addr_arg(func_addr, addr, arg_idx, opnum[, callee])` | Read numform JSON via call-arg coordinate |
| `set_numform_addr_expr(func_addr, addr, opnum, spec[, op_name[, nth]])` | Set/clear numform via expression coordinate |
| `get_numform_addr_expr(func_addr, addr, opnum[, op_name[, nth]])` | Read numform JSON via expression coordinate |

Decompiler local and label mutation is table-driven:
- List locals with `SELECT idx, name, type, comment, size, is_arg, is_result, stkoff, mreg FROM ctree_lvars WHERE func_addr = ... ORDER BY idx`.
- Rename or comment locals with `UPDATE ctree_lvars SET name = ...` or `comment = ...` using `func_addr` plus a selected `idx`.
- Rename labels with `UPDATE ctree_labels SET name = ... WHERE func_addr = ... AND label_num = ...`.

---

## File Generation

| Function | Description |
|----------|-------------|
| `gen_listing(path)` | Generate a full-database listing file (LST) |

```sql
SELECT gen_listing('C:/tmp/full.lst');
```

---

## Graph Generation

| Function | Description |
|----------|-------------|
| `gen_cfg_dot(addr)` | Generate CFG as DOT graph string |
| `gen_cfg_dot_file(addr, path)` | Write CFG DOT to file |
| `gen_schema_dot()` | Generate database schema as DOT |

```sql
SELECT gen_cfg_dot(0x401000);
SELECT gen_schema_dot();
```

---

## Database Persistence

| Function | Description |
|----------|-------------|
| `save_database()` | Persist the current IDA database file; returns `1` on success, `0` on failure |

```sql
-- Save once after a batch of SQL edits
SELECT save_database();
```

Use this in long-lived HTTP/MCP/plugin sessions when writes should persist.
Prefer `save_database()` over an IDAPython save snippet; it exercises idasql's
own persistence surface. `save_database()` can be costly on large databases, so
batch edits and save at an intentional boundary.

---

## Entity Search (grep)

Canonical workflow guidance lives in `../grep/SKILL.md`.

| Surface | Description |
|---------|-------------|
| `grep` table | Structured rows for composable SQL search |

```sql
SELECT name, kind, addr FROM grep WHERE pattern = 'sub%' LIMIT 10;
SELECT name, kind, addr FROM grep WHERE pattern = 'init' LIMIT 50 OFFSET 0;
```

---

## String List Functions

| Function | Description |
|----------|-------------|
| `rebuild_strings()` | Rebuild with ASCII + UTF-16, minlen 5 (default) |
| `rebuild_strings(minlen)` | Rebuild with custom minimum length |
| `rebuild_strings(minlen, types)` | Rebuild with custom length and type mask |

Type mask: `1`=ASCII, `2`=UTF-16, `4`=UTF-32, `3`=ASCII+UTF-16 (default), `7`=all.
Use `COUNT(*) FROM strings` for the current string-list count without materializing string rows.

```sql
SELECT COUNT(*) AS strings FROM strings;
SELECT rebuild_strings();
SELECT rebuild_strings(4);
SELECT rebuild_strings(5, 7);
```

---

## See Also

- `connect` — front-door catalog and routing matrix; use when picking a skill, not a function.
- The relevant per-topic skill for any function (`data` for bytes/strings, `decompiler` for `decompile()`, `disassembly` for `disasm_*`, `types` for `parse_decls`, `debugger` for patching helpers).

