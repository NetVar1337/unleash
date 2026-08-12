---
name: xrefs
description: "Use for IDA cross-refs: callers, callees, imports, data refs, call graphs."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: re
  upstream: C:\Users\Admin\.agents\skills\xrefs\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\xrefs\SKILL.md

## Trigger Intents

Use this skill when user asks:
- "Who calls this?" / "What does this call?"
- "Where is this string/import referenced?"
- "Show call graph dependencies."

Route to:
- `grep` for candidate entity lookup by name/pattern before relationship analysis
- `analysis` for broader triage context
- `decompiler` for semantic interpretation after graph narrowing
- `disassembly` for instruction-level call-site proof

---

## Do This First (Warm-Start Sequence)

```sql
-- 1) Core relation volume
SELECT COUNT(*) AS xref_count FROM xrefs;

-- 2) Top imports (dependency hints)
SELECT module, COUNT(*) AS import_count
FROM imports
GROUP BY module
ORDER BY import_count DESC;

-- 3) Most called functions
SELECT printf('0x%X', to_addr) AS callee, COUNT(*) AS callers
FROM xrefs
WHERE is_code = 1
GROUP BY to_addr
ORDER BY callers DESC
LIMIT 20;
```

Interpretation guidance:
- Use relation counts to prioritize hotspots before expensive deep analysis.
- Prefer indexed filters (`to_addr`/`from_addr`) for fast response.

---

## Failure and Recovery

- Full-scan query too slow:
  - Add `to_addr = X` or `from_addr = X` constraints.
- Target unresolved by name:
  - Resolve/verify address first through the `names` table or explicit EA literals.
- Sparse results:
  - Pivot through `imports`, `strings`, or `disasm_calls` joins.
- Looking for references to a struct/union FIELD:
  - Use `struct_member_xrefs` (filter by `type_name`/`type_ordinal`/`member_id`).
    Do NOT `LIKE`-scan `instructions.disasm` / `instruction_operands.text` for the
    field name — it is the slow, crash-adjacent pattern this table replaces.
- Target lives inside a static struct or dispatch table:
  - In modern binaries a string/data target is often referenced from a
    registration or vtable, not from code. `xrefs WHERE to_addr = X` may then
    return only `.rdata` rows. Walk back to the table head, then re-query
    xrefs against the *table's* address.
    ```sql
    -- 1) find the head of the item that contains the target
    SELECT addr FROM heads
    WHERE addr <= 0x140027D50 ORDER BY addr DESC LIMIT 1;
    -- 2) xrefs into the table head (where the code consumer points)
    SELECT * FROM xrefs WHERE to_addr = <head> AND is_code = 1;
    ```

---

## Handoff Patterns

1. `xrefs` -> `decompiler` for top candidate function semantics.
2. `xrefs` -> `analysis` for campaign-level synthesis.
3. `xrefs` -> `annotations` to persist relationship findings.

---

## xrefs
Cross-references - the most important table for understanding code relationships.

| Column | Type | Description |
|--------|------|-------------|
| `from_addr` | INT | Source address (who references) |
| `to_addr` | INT | Target address (what is referenced) |
| `type` | INT | Xref type code (never the ordinary-flow code, see below) |
| `is_code` | INT | 1=code xref (call/jump), 0=data xref |
| `from_func` | INT | Pre-computed containing function address (NULL when not in a function) |

Semantics: the surface exposes **real references only** — ordinary fall-through
flow edges (fl_F: each instruction "referencing" the next) are excluded on both
the full scan and the `to_addr`/`from_addr`/`from_func` fast paths, matching
IDA's Ctrl-X view and the other xsql tools. Counts therefore reflect
calls/jumps/data refs, not instruction adjacency.

```sql
-- Who calls function at 0x401000?
SELECT printf('0x%X', from_addr) as caller FROM xrefs WHERE to_addr = 0x401000 AND is_code = 1;

-- What does function at 0x401000 reference?
SELECT printf('0x%X', to_addr) as target FROM xrefs WHERE from_addr >= 0x401000 AND from_addr < 0x401100;
```

---

## struct_member_xrefs
Cross-references **to a specific struct/union member** (Issue 11). Resolves the
member's IDA tid and enumerates xrefs directly — use this instead of `LIKE`
scans over `instructions.disasm` / `instruction_operands.text`. **Requires a
filter** on `type_ordinal`, `type_name`, or `member_id` (an unfiltered query
errors). Embedded members are expanded recursively with a dotted `member_path`.

| Column | Type | Description |
|--------|------|-------------|
| `type_ordinal` / `type_name` | INT/TEXT | Queried (top-level) type |
| `member_index` | INT | Leaf index within its immediate parent type |
| `member_name` / `member_path` | TEXT | Leaf name / dotted path (`payload.leaf.data`) |
| `member_offset` / `member_offset_bits` | INT | Cumulative offset within the queried type |
| `member_id` | INT | Member tid (also a filter key) |
| `xref_from` | INT | Referencing address |
| `xref_to` | INT | Member tid (== `member_id`) |
| `xref_type` | INT | Raw IDA xref type code |
| `xref_kind` | TEXT | `read` / `write` / `offset` / `call` / `jump` / `flow` |
| `function_addr` / `function_name` | INT/TEXT | Containing function (NULL if none) |
| `operand_index` | INT | Operand referencing the member (best-effort; NULL) |
| `instruction_text` | TEXT | Disasm line at `xref_from` (NULL if not an instruction) |
| `access_offset` / `access_size` | INT | Access sub-offset / width (best-effort; NULL) |

```sql
-- Who writes _EH3_EXCEPTION_REGISTRATION.TryLevel?
SELECT member_path, xref_kind, function_name, instruction_text
FROM struct_member_xrefs
WHERE type_name = '_EH3_EXCEPTION_REGISTRATION' AND member_name = 'TryLevel';

-- All member references for a type (incl. nested), grouped by member
SELECT member_path, count(*) AS refs FROM struct_member_xrefs
WHERE type_ordinal = (SELECT ordinal FROM types WHERE name = 'db_entry_t')
GROUP BY member_path;
```

---

## imports
Imported functions from external libraries.

| Column | Type | Description |
|--------|------|-------------|
| `addr` | INT | Import address (IAT entry) |
| `name` | TEXT | Import name |
| `module` | TEXT | Module/DLL name |
| `ordinal` | INT | Import ordinal |
| `folder_path` | TEXT | Writable import folder path |
| `full_path` | TEXT | Full import dirtree path |

```sql
-- Imports from kernel32.dll
SELECT name FROM imports WHERE module LIKE '%kernel32%';

-- Organize network-related imports
UPDATE imports
SET folder_path = 'idasql/imports/network'
WHERE name LIKE '%socket%' OR name LIKE '%connect%' OR name LIKE '%send%';
```

---

## Convenience Views

### callers
Who calls each function. Caller names are resolved by the view from the `funcs` and `names` tables.

| Column | Type | Description |
|--------|------|-------------|
| `func_addr` | INT | Target function address |
| `caller_addr` | INT | Xref source address |
| `caller_name` | TEXT | Calling function name |
| `caller_func_addr` | INT | Calling function start (from `from_func`) |

Underlying query:
```sql
SELECT x.to_addr as func_addr, x.from_addr as caller_addr,
       COALESCE((SELECT name FROM names WHERE addr = x.from_func LIMIT 1), printf('sub_%X', x.from_func)) as caller_name,
       x.from_func as caller_func_addr
FROM xrefs x WHERE x.is_code = 1 AND x.from_func != 0
```

```sql
-- Who calls function at 0x401000?
SELECT caller_name, printf('0x%X', caller_addr) as from_addr
FROM callers WHERE func_addr = 0x401000;

-- Most called functions
SELECT printf('0x%X', func_addr) as addr, COUNT(*) as callers
FROM callers GROUP BY func_addr ORDER BY callers DESC LIMIT 10;
```

### callees
What each function calls. Inverse of callers view. Uses `from_func` for efficient function-level grouping.

| Column | Type | Description |
|--------|------|-------------|
| `func_addr` | INT | Calling function address (from `from_func`) |
| `func_name` | TEXT | Calling function name |
| `callee_addr` | INT | Called address |
| `callee_name` | TEXT | Called function/symbol name |

```sql
-- What does main call?
SELECT callee_name, printf('0x%X', callee_addr) as addr
FROM callees WHERE func_name LIKE '%main%';

-- Functions making most calls
SELECT func_name, COUNT(*) as call_count
FROM callees GROUP BY func_addr ORDER BY call_count DESC LIMIT 10;
```

### string_refs
Pre-joined view of string cross-references with containing function info.

| Column | Type | Description |
|--------|------|-------------|
| `string_addr` | INT | Address of the string |
| `string_value` | TEXT | String content |
| `string_length` | INT | String length |
| `ref_addr` | INT | Address of the referencing instruction |
| `func_addr` | INT | Containing function address |
| `func_name` | TEXT | Containing function name |

```sql
-- Strings referenced by a specific function
SELECT string_value, func_name FROM string_refs WHERE func_addr = 0x401000;

-- Find functions referencing password-related strings
SELECT string_value, func_name FROM string_refs WHERE string_value LIKE '%password%';

-- Most referenced strings
SELECT string_value, COUNT(*) as ref_count
FROM string_refs GROUP BY string_addr ORDER BY ref_count DESC LIMIT 10;
```

### data_refs
Cached table of data (non-code) cross-references with containing function info.
Whole-program aggregates are supported.

| Column | Type | Description |
|--------|------|-------------|
| `from_addr` | INT | Source address of the reference |
| `to_addr` | INT | Target data address |
| `from_func_addr` | INT | Containing function address |
| `from_func_name` | TEXT | Containing function name |
| `ref_type` | INT | Xref type code |

```sql
-- Data references from a specific function
SELECT * FROM data_refs WHERE from_func_addr = 0x401000;

-- Functions with most data references
SELECT from_func_name, COUNT(*) as data_ref_count
FROM data_refs GROUP BY from_func_addr ORDER BY data_ref_count DESC LIMIT 10;
```

---

## call_graph
Table-valued function for BFS call graph traversal. Uses HIDDEN parameters for traversal control.

| Column | Type | Description |
|--------|------|-------------|
| `func_addr` | INT | Function address in the graph |
| `func_name` | TEXT | Function name |
| `depth` | INT | BFS depth from start |
| `parent_addr` | INT | Parent function address in the traversal |
| `start` | INT | **HIDDEN** — Starting function address |
| `direction` | TEXT | **HIDDEN** — `'down'` (callees), `'up'` (callers), or `'both'` |
| `max_depth` | INT | **HIDDEN** — Maximum traversal depth |

**Always provide WHERE constraints for all 3 hidden parameters.**

```sql
-- Forward call tree from main
SELECT func_name, depth FROM call_graph
WHERE start = (SELECT addr FROM funcs WHERE name = 'main')
  AND direction = 'down' AND max_depth = 5;

-- All transitive callers
SELECT func_name, depth FROM call_graph
WHERE start = 0x405000 AND direction = 'up' AND max_depth = 10;

-- Bidirectional exploration
SELECT func_name, depth FROM call_graph
WHERE start = 0x401000 AND direction = 'both' AND max_depth = 3;

-- Join with string_refs to find strings reachable from a function
SELECT DISTINCT sr.string_value, sr.func_name
FROM call_graph cg
JOIN string_refs sr ON sr.func_addr = cg.func_addr
WHERE cg.start = 0x401000 AND cg.direction = 'down' AND cg.max_depth = 3;

-- Imported APIs reachable from a function's call tree
SELECT DISTINCT i.module, i.name as api
FROM call_graph cg
JOIN disasm_calls dc ON dc.func_addr = cg.func_addr
JOIN imports i ON dc.callee_addr = i.addr
WHERE cg.start = 0x401000 AND cg.direction = 'down' AND cg.max_depth = 5
ORDER BY i.module, i.name;
```

Performance: BFS with visited set. O(reachable functions). Always constrain hidden params.
Use this pattern when the destination is an import.

---

## shortest_path
Table-valued function for finding the shortest call path between two functions. Uses bidirectional BFS.

| Column | Type | Description |
|--------|------|-------------|
| `step` | INT | Step number in the path (0 = source) |
| `func_addr` | INT | Function address at this step |
| `func_name` | TEXT | Function name at this step |
| `from_addr` | INT | **HIDDEN** — Source function address |
| `to_addr` | INT | **HIDDEN** — Destination function address |
| `max_depth` | INT | **HIDDEN** — Maximum search depth |

**Always provide WHERE constraints for all 3 hidden parameters.**
Both endpoints must resolve to functions. Imported API addresses from `imports`
are not valid `shortest_path` endpoints.

```sql
-- Find shortest call path between two functions
SELECT step, func_name FROM shortest_path
WHERE from_addr = (SELECT addr FROM funcs WHERE name = 'main')
  AND to_addr = (SELECT addr FROM funcs WHERE name = 'target_func')
  AND max_depth = 20;

-- Check reachability between two functions
SELECT COUNT(*) > 0 as reachable FROM shortest_path
WHERE from_addr = 0x401000 AND to_addr = 0x405000 AND max_depth = 20;

-- Annotate path steps with call count and string refs
SELECT sp.step, sp.func_name,
       (SELECT COUNT(*) FROM disasm_calls dc WHERE dc.func_addr = sp.func_addr) as calls_made,
       (SELECT COUNT(*) FROM string_refs sr WHERE sr.func_addr = sp.func_addr) as strings_used
FROM shortest_path sp
WHERE sp.from_addr = 0x401000 AND sp.to_addr = 0x405000 AND sp.max_depth = 20
ORDER BY sp.step;
```

Performance: Bidirectional BFS. O(b^(d/2)) where b is branching factor and d is path length. Returns empty result set if no path exists within max_depth.

---

## Table-First Cross-Reference Queries

```sql
-- Incoming references to an address
SELECT from_addr, to_addr, type, is_code, from_func
FROM xrefs
WHERE to_addr = 0x401000;

-- Exact outgoing references from an item address
SELECT from_addr, to_addr, type, is_code, from_func
FROM xrefs
WHERE from_addr = 0x401000;

-- Outgoing references from anywhere inside a function
SELECT from_addr, to_addr, type, is_code, from_func
FROM xrefs
WHERE from_func = 0x401000;
```

---

## grep

Use `grep` to resolve internal symbols, types, and members before doing relationship analysis. Use `imports` when the callee may exist only as an imported API.
Canonical usage lives in `../grep/SKILL.md`.
For canonical schema and owner mapping, see `../connect/references/schema-catalog.md` (`grep`).

```sql
-- Resolve internal functions with grep
SELECT name, kind, addr
FROM grep
WHERE pattern = 'main%' AND kind = 'function'
ORDER BY name;

-- Resolve imported APIs with imports
SELECT module, name, addr
FROM imports
WHERE name LIKE 'CreateFile%'
ORDER BY module, name;

-- Then pivot into callers/callees/xrefs
SELECT caller_name, printf('0x%X', caller_addr) AS from_addr
FROM callers
WHERE func_addr = (
    SELECT addr
    FROM imports
    WHERE name = 'CreateFileW'
    ORDER BY name
    LIMIT 1
);
```

---

## Performance Rules

### Constraint Pushdown

The `xrefs` table has **optimized filters** using efficient IDA SDK APIs:

| Filter | Cost | Behavior |
|--------|------|----------|
| `to_addr = X` | 0.5 | O(xrefs to X) — fast, uses IDA's xref index |
| `from_addr = X` | 0.5 | O(xrefs from X) — fast, uses IDA's xref index |
| `from_func = X` | 1.0 | O(callees of X) — uses XrefsFromFuncIterator |
| No equality filter on `to_addr` / `from_addr` / `from_func` | — | Falls back to a cache of xrefs to function entry points only — avoid relying on it for complete import/data/non-function coverage |

**Always filter xrefs by `to_addr`, `from_addr`, or `from_func`. Avoid unconstrained scans.**

```sql
-- FAST: xrefs to a specific target
SELECT * FROM xrefs WHERE to_addr = 0x401000;

-- FAST: xrefs from a specific source
SELECT * FROM xrefs WHERE from_addr = 0x401000;

-- FAST: all xrefs originating from a function
SELECT * FROM xrefs WHERE from_func = 0x401000;

-- INCOMPLETE/avoid: unconstrained scan falls back to a function-entry cache
SELECT * FROM xrefs WHERE is_code = 1;
```

---

## Common Xref Patterns

### Find Most Called Functions

```sql
SELECT f.name, COUNT(*) as caller_count
FROM funcs f
JOIN xrefs x ON f.addr = x.to_addr
WHERE x.is_code = 1
GROUP BY f.addr
ORDER BY caller_count DESC
LIMIT 10;
```

### Find Functions Calling a Specific API

```sql
SELECT DISTINCT (SELECT name FROM funcs WHERE from_addr >= addr AND from_addr < end_addr LIMIT 1) as caller
FROM xrefs
WHERE to_addr = (SELECT addr FROM imports WHERE name = 'CreateFileW');
```

### String Cross-Reference Analysis

```sql
SELECT s.content, (SELECT name FROM funcs WHERE x.from_addr >= addr AND x.from_addr < end_addr LIMIT 1) as used_by
FROM strings s
JOIN xrefs x ON s.addr = x.to_addr
WHERE s.content LIKE '%password%';
```

### Import Dependency Map

```sql
-- Which modules does each function depend on?
SELECT f.name as func_name, i.module, COUNT(*) as api_count
FROM funcs f
JOIN disasm_calls dc ON dc.func_addr = f.addr
JOIN imports i ON dc.callee_addr = i.addr
GROUP BY f.addr, i.module
ORDER BY f.name, api_count DESC;
```

### Data Section References

```sql
-- Functions referencing data sections
SELECT
    f.name,
    s.name as segment,
    COUNT(*) as data_refs
FROM funcs f
JOIN xrefs x ON x.from_addr BETWEEN f.addr AND f.end_addr
JOIN segments s ON x.to_addr BETWEEN s.start_addr AND s.end_addr
WHERE s.class = 'DATA' AND x.is_code = 0
GROUP BY f.addr, s.name
ORDER BY data_refs DESC
LIMIT 20;
```

---

## Advanced Xref Patterns (CTEs and Recursive Queries)

> **Prefer `call_graph` over manual CTEs.** The `call_graph` table uses C++ BFS with a visited set — correct BFS depths, no duplicate expansion on diamond-shaped call graphs, and early termination.

**Before** (recursive CTE — no visited tracking, exponential on diamond graphs):
```sql
WITH RECURSIVE call_chain(root, current_func, depth) AS (
    SELECT 0x401000, callee_addr, 1
    FROM disasm_calls WHERE func_addr = 0x401000
    UNION ALL
    SELECT cc.root, dc.callee_addr, cc.depth + 1
    FROM call_chain cc
    JOIN disasm_calls dc ON dc.func_addr = cc.current_func
    WHERE cc.depth < 10
)
SELECT DISTINCT current_func, MIN(depth) FROM call_chain GROUP BY current_func;
```

**After** (call_graph table — C++ BFS with visited set, correct depths):
```sql
SELECT func_addr, func_name, depth FROM call_graph
WHERE start = 0x401000 AND direction = 'down' AND max_depth = 10;
```

### Recursive Call Graph — Forward Traversal

> **Note:** For targeted traversal, prefer the `call_graph` table which uses C++ BFS with visited tracking. Use recursive CTEs only when you need custom join logic (e.g., filtering by callee properties at each step).

Find all functions reachable from a starting function (up to depth 5):

```sql
-- Preferred: use call_graph table
SELECT func_name, depth FROM call_graph
WHERE start = (SELECT addr FROM funcs WHERE name = 'main')
  AND direction = 'down' AND max_depth = 5;

-- Manual CTE (when custom filtering is needed at each step):
WITH RECURSIVE cg AS (
    SELECT addr as func_addr, name, 0 as depth
    FROM funcs WHERE name = 'main'

    UNION ALL

    SELECT f.addr, f.name, cg.depth + 1
    FROM cg
    JOIN disasm_calls dc ON dc.func_addr = cg.func_addr
    JOIN funcs f ON f.addr = dc.callee_addr
    WHERE cg.depth < 5
      AND dc.callee_addr != 0
)
SELECT DISTINCT func_addr, name, MIN(depth) as min_depth
FROM cg
GROUP BY func_addr
ORDER BY min_depth, name;
```

### Recursive Call Graph — Reverse (Who Calls This?)

> **Note:** For targeted traversal, prefer the `call_graph` table with `direction = 'up'`. Use recursive CTEs only when you need custom join logic.

Trace callers transitively up to depth 5:

```sql
-- Preferred: use call_graph table
SELECT func_name, depth FROM call_graph
WHERE start = 0x401000 AND direction = 'up' AND max_depth = 5;

-- Manual CTE (when custom filtering is needed at each step):
WITH RECURSIVE callers_cte AS (
    SELECT DISTINCT dc.func_addr, 1 as depth
    FROM disasm_calls dc
    WHERE dc.callee_addr = 0x401000

    UNION ALL

    SELECT DISTINCT dc.func_addr, c.depth + 1
    FROM callers_cte c
    JOIN disasm_calls dc ON dc.callee_addr = c.func_addr
    WHERE c.depth < 5
)
SELECT (SELECT name FROM funcs WHERE func_addr >= addr AND func_addr < end_addr LIMIT 1) as caller, MIN(depth) as distance
FROM callers_cte
GROUP BY func_addr
ORDER BY distance, caller;
```

### CTE: Functions That Both Call malloc AND Check NULL

```sql
WITH malloc_callers AS (
    SELECT DISTINCT func_addr
    FROM disasm_calls
    WHERE callee_name LIKE '%malloc%'
),
null_checkers AS (
    SELECT DISTINCT func_addr
    FROM ctree_v_comparisons
    WHERE rhs_num = 0 AND op_name = 'cot_eq'
)
SELECT f.name
FROM funcs f
JOIN malloc_callers m ON f.addr = m.func_addr
JOIN null_checkers n ON f.addr = n.func_addr;
```

### CTE: Memory Allocation Without Free (Potential Leaks)

```sql
WITH allocators AS (
    SELECT func_addr, COUNT(*) as alloc_count
    FROM disasm_calls
    WHERE callee_name LIKE '%alloc%' OR callee_name LIKE '%malloc%'
    GROUP BY func_addr
),
freers AS (
    SELECT func_addr, COUNT(*) as free_count
    FROM disasm_calls
    WHERE callee_name LIKE '%free%'
    GROUP BY func_addr
)
SELECT f.name,
       COALESCE(a.alloc_count, 0) as allocations,
       COALESCE(r.free_count, 0) as frees
FROM funcs f
LEFT JOIN allocators a ON f.addr = a.func_addr
LEFT JOIN freers r ON f.addr = r.func_addr
WHERE a.alloc_count > 0 AND COALESCE(r.free_count, 0) = 0
ORDER BY allocations DESC;
```

### EXISTS: Functions With at Least One String Reference

More efficient than JOIN + DISTINCT for existence checks:

```sql
SELECT f.name
FROM funcs f
WHERE EXISTS (
    SELECT 1 FROM xrefs x
    JOIN strings s ON x.to_addr = s.addr
    WHERE x.from_addr BETWEEN f.addr AND f.end_addr
);
```

### EXISTS: Leaf Functions (No Outgoing Calls)

```sql
SELECT f.name, f.size
FROM funcs f
WHERE NOT EXISTS (
    SELECT 1 FROM disasm_calls dc
    WHERE dc.func_addr = f.addr
)
ORDER BY f.size DESC;
```

---

## See Also

- `disassembly` — call-site context (`disasm_at`, `disasm_calls`, operand-level reads).
- `data` — string/data targets behind a `to_addr`; the canonical pivot when xrefs returns `.rdata`-only rows.
- `decompiler` — calling-code logic and indirect-call resolution via callee types.

