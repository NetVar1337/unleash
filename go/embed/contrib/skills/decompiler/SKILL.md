---
name: decompiler
description: "Use for IDA pseudocode, ctree AST, locals, labels, decompiler-driven cleanup."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: re
  upstream: C:\Users\Admin\.agents\skills\decompiler\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\decompiler\SKILL.md

## Trigger Intents

Use this skill when user asks for:
- "decompile this function"
- pseudocode understanding or AST-level analysis
- local variable semantics in decompiled form
- decompiler-centric pattern mining (returns/calls/conditions)

Route to:
- `annotations` for persistent comments/renames after interpretation
- `types` for struct/enum/type construction and application
- `disassembly` when decompiler is unavailable or insufficient

---

## Do This First (Warm-Start Sequence)

```sql
-- 1) Capability/profile probe
SELECT * FROM pragma_table_list WHERE name IN ('pseudocode', 'ctree', 'ctree_lvars');

-- 2) Pick one concrete function target
SELECT name, printf('0x%X', addr) AS addr, size
FROM funcs
ORDER BY size DESC
LIMIT 10;

-- 3) View decompiled text via primary read surface
SELECT decompile(0x401000);
```

Interpretation guidance:
- `decompile(addr)` is primary display surface.
- `pseudocode`/`ctree*` are structured query/edit surfaces.

---

## Global Constraint Reminder (Critical)

Always constrain decompiler tables by function:

```sql
WHERE func_addr = 0x...
```

Without this, decompiler tables may decompile every function and become extremely slow.

---

## Failure and Recovery

- No Hex-Rays/decompiler tables unavailable:
  - Fall back to `disassembly` + `xrefs` workflows.
- Empty/partial rows:
  - Confirm target `func_addr` exists and refresh decompile cache (`decompile(addr, 1)` where supported).
- Mutation did not appear:
  - Run mandatory mutation loop (read -> edit -> refresh -> verify).

---

## Handoff Patterns

1. `decompiler` -> `types` for local type seeding and richer declarations.
2. `decompiler` -> `annotations` for persistent narrative and naming.
3. `decompiler` -> `disassembly` for opcode-level validation.

---

## Decompiler Tables (Hex-Rays Required)

**CRITICAL:** Always filter by `func_addr`. Without constraint, these tables will decompile EVERY function - extremely slow!

### pseudocode
The `pseudocode` table is a structured line-by-line pseudocode with writable comments. **Use `decompile(addr)` to view pseudocode; use this table only for surgical edits (comments) or structured queries.**

| Column | Type | Writable | Description |
|--------|------|----------|-------------|
| `func_addr` | INT | No | Function address |
| `line_num` | INT | No | Line number |
| `line` | TEXT | No | Pseudocode text |
| `addr` | INT | No | Corresponding assembly address (from COLOR_ADDR anchor) |
| `comment` | TEXT | **Yes** | Decompiler comment at this addr |
| `comment_placement` | TEXT | **Yes** | Comment placement: `semi` (inline, default), `block1` (above line) |

Filter behavior:
- `WHERE func_addr = X`: best performance; iterates pseudocode for one function only.
- `WHERE addr = X`: decompiles only the containing function and returns matching lines for that EA.
- `WHERE line_num = N`: scans functions and returns rows at that line index; use only when you need cross-function line alignment.

**Comment placements:** `semi` (after `;`), `block1` (own line above), `block2`, `curly1`, `curly2`, `brace1`, `brace2`, `colon`, `case`, `else`, `do`, `asm`, `try`. An unrecognized placement string is silently coerced to `semi`.

```sql
-- VIEWING: Use decompile() function, NOT the pseudocode table
SELECT decompile(0x401000);

-- COMMENTING: Use pseudocode table to add/edit/delete comments
UPDATE pseudocode SET comment_placement = 'semi',
                      comment = 'buffer overflow here'
WHERE func_addr = 0x401000 AND addr = 0x401020;

-- Add block comment (appears on own line above the statement)
UPDATE pseudocode SET comment_placement = 'block1', comment = 'vulnerable call'
WHERE func_addr = 0x401000 AND addr = 0x401020;

-- Delete comments at a resolved unique anchor
UPDATE pseudocode SET comment = NULL
WHERE func_addr = 0x401000 AND addr = 0x401020;
```

True function comments are not part of `pseudocode`:
- use `UPDATE funcs SET comment = ... WHERE addr = ...` for the regular function comment
- use `UPDATE funcs SET rpt_comment = ... WHERE addr = ...` for the repeatable function comment

### pseudocode_orphan_comments
Persisted Hex-Rays comments that no longer attach to the current decompiled output of a live function. Use it to inspect or delete stale comments.

| Column | Type | Writable | Description |
|--------|------|----------|-------------|
| `func_addr` | INT | No | Function address |
| `func_name` | TEXT | No | Current function name for triage |
| `addr` | INT | No | Stored orphan comment EA |
| `comment_placement` | TEXT | No | Stored `treeloc_t.itp` placement |
| `orphan_comment` | TEXT | **Delete-only** | Stored orphan comment text |

Rules:
- `UPDATE ... SET orphan_comment = NULL` or `''` deletes that orphan comment.
- Any non-empty write is rejected.

### pseudocode_v_orphan_comment_groups
Grouped, read-only orphan triage surface. One row per function with orphan comments.

Columns: `func_addr`, `func_name`, `orphan_count`, `orphan_comments_json`

### Comment Anchor Resolution (Critical)

Use this recipe before writing heading-style decompiler notes.

Rules:
- Do not assume `addr == func_addr`.
- The first displayed pseudocode row often has `addr = 0` and is not the right write target.
- One `addr` can map to multiple rows (`{`, statement, `}`); prefer a unique non-brace anchor.
- For true function comments, update `funcs.comment` / `funcs.rpt_comment` instead of `pseudocode`.

```sql
-- Resolve the first attachable non-brace row near function start
SELECT line_num, addr, line
FROM pseudocode
WHERE func_addr = 0x401000
  AND addr != 0
  AND TRIM(line) NOT IN ('{', '}')
  AND addr IN (
    SELECT addr
    FROM pseudocode
    WHERE func_addr = 0x401000 AND addr != 0
    GROUP BY addr
    HAVING COUNT(*) = 1
  )
ORDER BY line_num
LIMIT 1;

-- Write a heading-style summary using the resolved addr
UPDATE pseudocode
SET comment_placement = 'block1',
    comment = 'One-paragraph summary of the function.'
WHERE func_addr = 0x401000
  AND addr = (
    SELECT addr
    FROM pseudocode
    WHERE func_addr = 0x401000
      AND addr != 0
      AND TRIM(line) NOT IN ('{', '}')
      AND addr IN (
        SELECT addr
        FROM pseudocode
        WHERE func_addr = 0x401000 AND addr != 0
        GROUP BY addr
        HAVING COUNT(*) = 1
      )
    ORDER BY line_num
    LIMIT 1
  );
```

### ctree
Full Abstract Syntax Tree of decompiled code.

| Column | Type | Description |
|--------|------|-------------|
| `func_addr` | INT | Function address |
| `item_id` | INT | Unique node ID |
| `is_expr` | INT | 1=expression, 0=statement |
| `op` | INT | Raw numeric op code |
| `op_name` | TEXT | Node type (`cot_call`, `cit_if`, etc.) |
| `addr` | INT | Address in binary |
| `parent_id` | INT | Parent node ID |
| `depth` | INT | Tree depth |
| `x_id`, `y_id`, `z_id` | INT | Child node IDs |
| `cond_id` | INT | Condition child ID (if/while/for) |
| `then_id` | INT | Then-branch child ID (if) |
| `else_id` | INT | Else-branch child ID (if) |
| `body_id` | INT | Body child ID (loops) |
| `init_id` | INT | Init child ID (for) |
| `step_id` | INT | Step child ID (for) |
| `var_idx` | INT | Local variable index |
| `obj_addr` | INT | Target address |
| `num_value` | INT | Numeric literal |
| `str_value` | TEXT | String literal |
| `helper_name` | TEXT | Helper function name (`cot_helper`) |
| `member_offset` | INT | Member offset (`cot_memptr`/`cot_memref`) |
| `var_name` | TEXT | Variable name |
| `var_is_stk` | INT | 1=variable is a stack variable |
| `var_is_reg` | INT | 1=variable is a register variable |
| `var_is_arg` | INT | 1=variable is a function argument |
| `obj_name` | TEXT | Symbol name |
| `label_num` | INT | Label number when node defines a label |
| `goto_label_num` | INT | Target label number for `cit_goto` nodes |

### ctree_lvars
Local variables from decompilation.

| Column | Type | Description |
|--------|------|-------------|
| `func_addr` | INT | Function address |
| `idx` | INT | Variable index |
| `name` | TEXT | Variable name |
| `type` | TEXT | Type string |
| `comment` | TEXT | Local-variable comment shown next to declaration |
| `size` | INT | Size in bytes |
| `is_arg` | INT | 1=function argument |
| `is_result` | INT | 1=holds the function result |
| `is_stk_var` | INT | 1=stack variable |
| `is_reg_var` | INT | 1=register variable |
| `stkoff` | INT | Stack offset |
| `mreg` | INT | Microcode register number (register variables) |

Mutation guidance:
- Prefer `idx`-based updates for deterministic writes.
- `type` accepts scalars, pointers, **and array types** — e.g. `SET type = 'WCHAR[6]'` or `'_BYTE[392]'` for wide stack strings / fixed buffers (the type must fit the local's stack slot).
- `comment` updates map to Hex-Rays local-variable comments (`lv.cmt`) and appear in `decompile(...)` output.

### ctree_labels
Decompiler control-flow labels. Supports UPDATE (`name`) and mirrors label facilities on `cfunc_t`.

| Column | Type | RW | Description |
|--------|------|----|-------------|
| `func_addr` | INT | R | Function address |
| `label_num` | INT | R | Label number (`LABEL_<n>`) |
| `name` | TEXT | RW | Current label name |
| `item_id` | INT | R | Backing ctree item id for this label |
| `item_addr` | INT | R | Address of label-bearing ctree item |
| `is_user_defined` | INT | R | 1 if name differs from default `LABEL_<n>` |

### ctree_call_args
Flattened call arguments for easy querying.

| Column | Type | Description |
|--------|------|-------------|
| `func_addr` | INT | Function address |
| `call_item_id` | INT | Call node ID |
| `call_addr` | INT | Call-site EA |
| `call_obj_name` | TEXT | Callee object name |
| `call_helper_name` | TEXT | Callee helper name |
| `arg_idx` | INT | Argument index (0-based) |
| `arg_item_id` | INT | Argument expression item ID |
| `arg_op` | TEXT | Argument type |
| `arg_var_idx` | INT | Local variable index when the argument is a variable |
| `arg_var_name` | TEXT | Variable name if applicable |
| `arg_var_is_stk` | INT | 1=argument variable is a stack variable |
| `arg_var_is_arg` | INT | 1=argument variable is itself a function argument |
| `arg_obj_addr` | INT | Object address when the argument is a global object |
| `arg_obj_name` | TEXT | Object name when the argument is a global object |
| `arg_num_value` | INT | Numeric value |
| `arg_str_value` | TEXT | String value |

---

## Decompiler Views

Pre-built views for common patterns (always filter by `func_addr`):

| View | Purpose |
|------|---------|
| `ctree_v_calls` | Function calls with callee info |
| `ctree_v_indirect_calls` | Indirect/dynamic call sites for call-site typing |
| `ctree_v_loops` | for/while/do loops |
| `ctree_v_ifs` | if statements |
| `ctree_v_comparisons` | Comparisons with operands |
| `ctree_v_signed_ops` | Signed comparison/arithmetic operations |
| `ctree_v_assignments` | Assignments with operands |
| `ctree_v_derefs` | Pointer dereferences |
| `ctree_v_returns` | Return statements with value details |
| `ctree_v_calls_in_loops` | Calls inside loops (recursive) |
| `ctree_v_calls_in_ifs` | Calls inside if branches (recursive) |
| `ctree_v_leaf_funcs` | Functions with no outgoing calls |
| `ctree_v_call_chains` | Call chain paths up to depth 10 |

---

## Type Tables and Views

For `types`, `types_members`, `types_enum_values`, `types_func_args` schemas, type views, and type CRUD examples, see `types` skill.

---

## SQL Functions — Decompilation

**When to use `decompile()` vs `pseudocode` table:**
- **Read/show pseudocode** -> always start with `SELECT decompile(addr)`. Returns full function as one text block with per-line prefixes.
- **Local declaration hints** -> declaration lines include compact local-variable index hints (`[lv:N]`) so rename operations can target `UPDATE ctree_lvars ... WHERE func_addr = ... AND idx = N` safely.
- **Need fresh output after edits** -> use `SELECT decompile(addr, 1)` to force re-decompilation.
- **Need structured line access or comment CRUD** -> query/update the `pseudocode` table.

| Function | Description |
|----------|-------------|
| `decompile(addr)` | **PREFERRED** -- Full pseudocode with line prefixes |
| `decompile(addr, 1)` | Same output but forces re-decompilation |
| `call_arg_addrs(call_addr)` | Read persisted argument-loader addresses as JSON |
| `set_union_selection(func_addr, addr, path)` | Set/clear union selection path at EA |
| `set_union_selection_item(func_addr, item_id, path)` | Set/clear union selection path by `ctree.item_id` |
| `set_union_selection_addr_arg(func_addr, addr, arg_idx, path[, callee])` | **PREFERRED** call-arg targeting helper |
| `call_arg_item(func_addr, addr, arg_idx[, callee])` | Resolve call-arg coordinate to explicit `arg_item_id` |
| `ctree_item_at(func_addr, addr[, op_name[, nth]])` | Resolve generic expression coordinate to explicit `ctree.item_id` |
| `set_union_selection_addr_expr(func_addr, addr, path[, op_name[, nth]])` | Set/clear union selection via generic expression coordinate |
| `get_union_selection(func_addr, addr)` | Read union selection path JSON at EA |
| `get_union_selection_item(func_addr, item_id)` | Read union selection path JSON by `ctree.item_id` |
| `get_union_selection_addr_arg(func_addr, addr, arg_idx[, callee])` | Read union selection JSON via call-arg coordinate |
| `get_union_selection_addr_expr(func_addr, addr[, op_name[, nth]])` | Read union selection JSON via generic expression coordinate |
| `set_numform(func_addr, addr, opnum, spec)` | Set/clear numform directly by EA + operand index |
| `get_numform(func_addr, addr, opnum)` | Read numform JSON directly by EA + operand index |
| `set_numform_item(func_addr, item_id, opnum, spec)` | Set/clear numform by explicit ctree item id |
| `get_numform_item(func_addr, item_id, opnum)` | Read numform JSON by explicit ctree item id |
| `set_numform_addr_arg(func_addr, addr, arg_idx, opnum, spec[, callee])` | Set/clear numform via call-arg coordinate |
| `get_numform_addr_arg(func_addr, addr, arg_idx, opnum[, callee])` | Read numform JSON via call-arg coordinate |
| `set_numform_addr_expr(func_addr, addr, opnum, spec[, op_name[, nth]])` | Set/clear numform via generic expression coordinate |
| `get_numform_addr_expr(func_addr, addr, opnum[, op_name[, nth]])` | Read numform JSON via generic expression coordinate |

Targeting guidance:
- Use `*_addr_arg` helpers for repeated callees and call-site arguments.
- Use `ctree_item_at(..., op_name, nth)` plus `*_addr_expr` helpers for non-call expressions and assignment-side struct/union population stores.

---

## SQL Functions — Modification

For `applied_types`, `parse_decls()`, and name writes via `names`/`funcs`, see `types` skill.

Preferred SQL write surface for function metadata:
- `UPDATE funcs SET name = '...', prototype = '...', comment = '...', rpt_comment = '...' WHERE addr = ...`
- `prototype` maps to `applied_types` behavior and invalidates decompiler cache.
- `comment` / `rpt_comment` map to `get_func_cmt()` / `set_func_cmt()`.

---

## Performance Rules

| Table | Architecture | Key Constraint | Notes |
|-------|-------------|----------------|-------|
| `pseudocode` | Cached | `func_addr` | Lazy per-function cache, freed after query |
| `pseudocode_orphan_comments` | Cached | `func_addr` | Query-scoped orphan rows; writable delete-only |
| `pseudocode_v_orphan_comment_groups` | Cached | `func_addr` | Query-scoped grouped orphan triage; start broad with `LIMIT` |
| `ctree` | Generator | `func_addr` | Lazy streaming, never materializes full result, respects LIMIT |
| `ctree_lvars` | Cached | `func_addr` | Lazy per-function cache, freed after query |
| `ctree_call_args` | Generator | `func_addr` | Lazy streaming, respects LIMIT |

**Critical rules:**
- **ALL decompiler tables require `func_addr` constraint.** Without it, every function is decompiled.
- Generator tables (`ctree`, `ctree_call_args`) stream rows lazily and stop at LIMIT.
- Decompiler views (`ctree_v_calls`, `ctree_v_indirect_calls`, `ctree_v_loops`, etc.) inherit the `func_addr` constraint -- always filter.
- **Hex-Rays cfunc cache:** `decompile(addr)` is internally cached. `decompile(addr, 1)` forces a full re-decompilation -- only use when you need to see effects of a mutation.

**Cost model:**
```
decompile(addr)          -> ~50-200ms first call, ~0ms cached
decompile(addr, 1)       -> ~50-200ms always (forces re-decompile)
ctree WHERE func_addr=X  -> one decompilation + streaming rows
ctree (no constraint)    -> one decompilation per row in funcs
```

---

## Additional Resources

- For detailed workflows (capability probing, mutation loop, call-site typing, local type seeding, fallback patterns, full worked examples): [references/decompiler-workflows.md](references/decompiler-workflows.md)
- For detailed view schemas (ctree_v_indirect_calls, ctree_v_returns): [references/decompiler-views.md](references/decompiler-views.md)
- For ctree node types, manipulation patterns, and advanced CTEs: [references/ctree-manipulation.md](references/ctree-manipulation.md)

---

## See Also

- `data` — raw bytes and string content referenced by decompiled code (the `bytes` table for per-byte reads and bounded-window reads via `WHERE start_addr = X AND n = N`; `hex(blob_concat(value))` for hex output).
- `disassembly` — instruction-level ground truth for the same function (`disasm_func`, `instructions`, `disasm_calls`).
- `xrefs` — callers and callees of a decompiled function; pivot from pseudocode to call graph.
- `types` — applied prototypes drive ctree shape; retype via `applied_types` / `funcs.prototype` to clean up casts and indirect calls.

