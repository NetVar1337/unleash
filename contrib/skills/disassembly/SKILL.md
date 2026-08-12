---
name: disassembly
description: "Use for IDA disassembly queries: functions, segments, instructions, blocks, operands, CFG."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: languages
  upstream: C:\Users\Admin\.agents\skills\disassembly\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\disassembly\SKILL.md

## Trigger Intents

Use this skill when user asks for:
- Function/segment/instruction inspection
- Call-site or control-flow analysis from disassembly
- Operand formatting and low-level code structure
- Raw byte/instruction-level evidence

Route to:
- `decompiler` for AST/pseudocode semantics
- `xrefs` for relationship-heavy caller/callee workflows
- `debugger` for patching/breakpoint actions

---

## Do This First (Warm-Start Sequence)

```sql
-- 1) Orientation
SELECT * FROM binary;

-- 2) Segment map
SELECT name, printf('0x%X', start_addr) AS start_addr, printf('0x%X', end_addr) AS end_addr, perm
FROM segments
ORDER BY start_addr;

-- 3) Largest functions (triage anchors)
SELECT name, printf('0x%X', addr) AS addr, size
FROM funcs
ORDER BY size DESC
LIMIT 20;
```

Interpretation guidance:
- Start from executable segments and largest/highly connected functions.
- Use `func_addr` constraints early when querying instruction-heavy surfaces.

---

## Failure and Recovery

- Slow queries on `instructions`/`heads`:
  - Add `WHERE func_addr = X` or tight EA ranges.
- Missing expected symbol names:
  - Pivot to address-based workflows and enrich via `names` updates later.
- Ambiguous control-flow behavior:
  - Cross-check with `disasm_calls` and then escalate to `decompiler`.

---

## Handoff Patterns

1. `disassembly` -> `xrefs` for relation expansion.
2. `disassembly` -> `decompiler` for semantic interpretation.
3. `disassembly` -> `debugger` for patch/breakpoint execution.

---

## Entity Tables

### funcs
All detected functions in the binary with prototype information.

| Column | Type | Description |
|--------|------|-------------|
| `addr` | INT | Function start address |
| `name` | TEXT | Function name |
| `size` | INT | Function size in bytes |
| `end_addr` | INT | Function end address |
| `flags` | INT | Function flags |
| `folder_path` | TEXT | RW function folder relative to Function window root; `NULL`/`''` means root |
| `full_path` | TEXT | RO full dirtree path, including function name |

**Prototype columns** (populated when type info available):

| Column | Type | Description |
|--------|------|-------------|
| `return_type` | TEXT | Return type string (e.g., "int", "void *") |
| `return_is_ptr` | INT | 1 if return type is pointer |
| `return_is_int` | INT | 1 if return type is exactly int |
| `return_is_integral` | INT | 1 if return type is int-like (int, long, DWORD, BOOL) |
| `return_is_void` | INT | 1 if return type is void |
| `arg_count` | INT | Number of function arguments |
| `calling_conv` | TEXT | Calling convention (cdecl, stdcall, fastcall, etc.) |

```sql
-- 10 largest functions
SELECT name, size FROM funcs ORDER BY size DESC LIMIT 10;

-- Functions starting with "sub_" (auto-named, not analyzed)
SELECT name, printf('0x%X', addr) as addr FROM funcs WHERE name LIKE 'sub_%';

-- Functions returning integers with 3+ arguments
SELECT name, return_type, arg_count FROM funcs
WHERE return_is_integral = 1 AND arg_count >= 3;

-- Functions organized by IDASQL annotation folders
SELECT addr, name, folder_path
FROM funcs
WHERE folder_path LIKE 'idasql/%'
ORDER BY folder_path, name;
```

**Write operations:**
```sql
-- Create a function
INSERT INTO funcs (addr) VALUES (0x401000);

-- Rename a function
UPDATE funcs SET name = 'my_func' WHERE addr = 0x401000;

-- Move a function into an IDA Function window folder
INSERT INTO dirtree_folders(tree, path) VALUES ('funcs', 'idasql/triage/network');
UPDATE funcs SET folder_path = 'idasql/triage/network' WHERE addr = 0x401000;

-- Move it back to root
UPDATE funcs SET folder_path = NULL WHERE addr = 0x401000;

-- Delete a function
DELETE FROM funcs WHERE addr = 0x401000;
```

Folder paths are relative `/` paths. IDASQL rejects `.`/`..`, duplicate separators, backslashes, non-empty folder deletes, and folder renames whose destination already exists.

### segments
Memory segments. Full CRUD: INSERT a region, DELETE it, and UPDATE `start_addr`
(rebase/move), `end_addr` (resize), `name`, `class`, `perm`.

| Column | Type | RW | Description |
|--------|------|----|-------------|
| `start_addr` | INT | RW | Segment start; UPDATE rebases (moves) the segment |
| `end_addr` | INT | RW | Segment end; UPDATE resizes the segment |
| `name` | TEXT | RW | Segment name (.text, .data, etc.) |
| `class` | TEXT | RW | Segment class (CODE, DATA) |
| `perm` | INT | RW | Permissions (R=4, W=2, X=1) |

```sql
-- Find executable segments
SELECT name, printf('0x%X', start_addr) as start FROM segments WHERE perm & 1 = 1;

-- Rename a segment
UPDATE segments SET name = '.mytext' WHERE start_addr = 0x401000;

-- Add a region (e.g. SRAM for a raw firmware image), then rebase it
INSERT INTO segments(start_addr, end_addr, name, class, perm)
  VALUES (0x20000000, 0x20005000, 'SRAM', 'DATA', 6);
UPDATE segments SET start_addr = 0x08000000 WHERE name = '.text';  -- rebase
```

### names
All named locations (functions, labels, data). Supports INSERT, UPDATE, and DELETE.

| Column | Type | RW | Description |
|--------|------|----|-------------|
| `addr` | INT | R | Address |
| `name` | TEXT | RW | Name |
| `folder_path` | TEXT | RW | Name-folder path; `NULL` means root |
| `full_path` | TEXT | R | Full name dirtree path |

```sql
-- Create/set a name
INSERT INTO names (addr, name) VALUES (0x401000, 'my_symbol');

-- Rename
UPDATE names SET name = 'my_symbol_renamed' WHERE addr = 0x401000;

-- Organize globals/labels into IDA name folders
UPDATE names SET folder_path = 'idasql/names/globals' WHERE name LIKE 'g_%';
UPDATE names SET folder_path = NULL WHERE addr = 0x401000;
```

### entries
Entry points (exports, program entry, tls callbacks, etc.).

| Column | Type | Description |
|--------|------|-------------|
| `ordinal` | INT | Export ordinal |
| `addr` | INT | Entry address |
| `name` | TEXT | Entry name |

---

## Instruction Tables

### instructions

`instructions` is the disassembly table. For scalar disassembly text at a specific EA, use `disasm_at(addr[, context])`.
Use `disasm_func()` or `disasm_range()` when you explicitly need a function/range listing.
Decoded instructions support DELETE (converts instruction to unexplored bytes) and operand representation updates via `operand*_format_spec`.

`WHERE func_addr = X` is the fast path (function-item iterator). Without it, the table scans all code heads.

| Column | Type | Description |
|--------|------|-------------|
| `addr` | INT | Instruction address |
| `func_addr` | INT | Containing function |
| `itype` | INT | Instruction type (architecture-specific) |
| `mnemonic` | TEXT | Instruction mnemonic |
| `size` | INT | Instruction size |
| `operand0..operand7` | TEXT | Operand text (`0..7`) |
| `disasm` | TEXT | Full disassembly line |
| `operand0_class..operand7_class` | TEXT | Operand class: `reg`, `imm`, `displ`, `mem`, ... |
| `operand0_repr_kind..operand7_repr_kind` | TEXT | Current representation: `plain`, `hex`, `dec`, `oct`, `bin`, `char`, `float`, `enum`, `offset`, `stroff`, `sizeof`, `segment`, `stkvar`, `forced` |
| `operand0_repr_type_name..operand7_repr_type_name` | TEXT | Enum/stroff/sizeof type, offset base (hex), or forced text |
| `operand0_format_spec..operand7_format_spec` | TEXT (RW) | Apply/clear representation for a specific operand (see vocabulary below) |

```sql
-- Instruction profile of a function (FAST)
SELECT mnemonic, COUNT(*) as count
FROM instructions WHERE func_addr = 0x401330
GROUP BY mnemonic ORDER BY count DESC;

-- Find all call instructions in a function
SELECT addr, disasm FROM instructions
WHERE func_addr = 0x401000 AND mnemonic = 'call';

-- Apply enum representation to operand 1
UPDATE instructions
SET operand1_format_spec = 'enum:MY_ENUM'
WHERE addr = 0x401020;

-- Apply struct-offset representation to operand 0
-- e.g. makes `[rax+10h]` display as `[rax+command_t.field_name]`
UPDATE instructions
SET operand0_format_spec = 'stroff:command_t'
WHERE addr = 0x140001BE8;

-- Struct-offset with a base delta (subtracted before member resolution)
UPDATE instructions
SET operand0_format_spec = 'stroff:command_t,delta=16'
WHERE addr = 0x140001BE8;

-- Nested member path: separate type names with '/'
UPDATE instructions
SET operand0_format_spec = 'stroff:outer_t/inner_t'
WHERE addr = 0x140001BE8;

-- Clear representation back to plain
UPDATE instructions
SET operand1_format_spec = 'clear'
WHERE addr = 0x401020;
```

The format-spec is verified after apply: the UPDATE re-reads the operand and
fails (as a SQL error) if the resulting `repr_kind`, type path, or delta don't
match what was requested. Read back `operandN_repr_kind` /
`operandN_repr_type_name` to confirm.

**Full `operandN_format_spec` vocabulary** (all disassembly-level, no IDAPython
needed):

| Spec | Effect |
|---|---|
| `hex` `dec` `oct` `bin` | number base (radix) |
| `char` | character constant |
| `float` | floating-point |
| `offset` | plain offset, base 0 |
| `offset:<base>` | offset with a user-defined base (`<base>` = symbol name or address) |
| `enum:<NAME>[,serial=N]` / `enum:<NAME>::<MEMBER>` | enum constant |
| `stroff:<Type[/Nested...]>[,delta=N]` | struct-member offset |
| `sizeof:<STRUCT>` | struct-size constant — renders `size STRUCT` |
| `segment` | segment selector |
| `stkvar` | stack variable (operand must be a stack reference) |
| `forced:<text>` | forced (manual) operand text, e.g. `forced:5 shl 3` |
| `clear` / `plain` / `none` | revert to default representation |

Combinable modifiers (suffix on any base, or standalone to modify the current
representation): `,signed` / `,unsigned` (toggle negative display) and
`,bnot` / `,nobnot` (toggle bitwise-not display). Example: `dec,signed`,
`hex,bnot`, or just `signed`.

```sql
-- Number base, character, and sign/bitwise-not modifiers
UPDATE instructions SET operand1_format_spec = 'hex'        WHERE addr = 0x401020;
UPDATE instructions SET operand1_format_spec = 'dec,signed' WHERE addr = 0x401020;
UPDATE instructions SET operand1_format_spec = 'char'       WHERE addr = 0x401020;

-- Offsets (plain and user-defined base)
UPDATE instructions SET operand1_format_spec = 'offset'            WHERE addr = 0x401020;
UPDATE instructions SET operand1_format_spec = 'offset:tbl_start'  WHERE addr = 0x401020;

-- sizeof: an immediate equal to sizeof(STRUCT) renders as `size STRUCT`
UPDATE instructions SET operand1_format_spec = 'sizeof:FILEDEF' WHERE addr = 0x4015EB;

-- Forced operand text
UPDATE instructions SET operand1_format_spec = 'forced:5 shl 3' WHERE addr = 0x401020;
```

### instruction_operands
One row per decoded non-void operand. Use this table for operand type/value details and for joinable replacements of old operand/decode helper functions.

| Column | Type | Description |
|--------|------|-------------|
| `addr` | INT | Instruction address |
| `func_addr` | INT | Containing function |
| `opnum` | INT | Operand index |
| `text` | TEXT | Operand text |
| `type_code` | INT | IDA operand type code |
| `type_name` | TEXT | Operand type name (`reg`, `imm`, `near`, ...) |
| `dtype` | INT | Operand dtype |
| `reg` | INT | Register number when applicable |
| `op_addr` | INT | Referenced address/displacement when applicable |
| `raw_value` | INT | Raw operand value |
| `value` | INT | Best-effort scalar operand value |

> **Note (addr vs op_addr):** `addr` is the **instruction** address; the operand's
> **referenced** address is `op_addr`. If you want the target an operand points at
> (e.g. a jump/call/data reference), select `op_addr` — `addr` identifies the
> instruction the operand belongs to, not what it references.

```sql
SELECT opnum, text, type_name, value
FROM instruction_operands
WHERE addr = 0x401000
ORDER BY opnum;

SELECT i.addr, i.itype, i.mnemonic, o.opnum, o.text, o.type_name, o.value
FROM instructions i
LEFT JOIN instruction_operands o
  ON o.addr = i.addr AND o.func_addr = 0x401000
WHERE i.func_addr = 0x401000
ORDER BY i.addr, o.opnum;
```

**Performance:** `WHERE addr = X` decodes one instruction; `WHERE func_addr = X` uses O(function_size) iteration. Without one of these constraints, it scans the entire database.

### disasm_calls
All call instructions with resolved targets and optional call-site prototype overrides.

| Column | Type | Description |
|--------|------|-------------|
| `func_addr` | INT | Function containing the call |
| `addr` | INT | Call instruction address |
| `callee_addr` | INT | Target address (0 if unknown) |
| `callee_name` | TEXT | Target name |
| `callee_type` | TEXT | RW nullable call-site prototype; `UPDATE` applies/replaces, `NULL` or empty clears |

```sql
-- Functions that call malloc
SELECT DISTINCT (SELECT name FROM funcs WHERE func_addr >= addr AND func_addr < end_addr LIMIT 1) as caller
FROM disasm_calls WHERE callee_name LIKE '%malloc%';

-- Apply/replace a call-site prototype, then clear it
UPDATE disasm_calls
SET callee_type = 'int (__fastcall *)(const char *path)'
WHERE addr = 0x401234;
UPDATE disasm_calls SET callee_type = NULL WHERE addr = 0x401234;
```

Applying `callee_type` on a **direct** call preserves the real callee symbol, so
re-decompiling the caller afterward is safe (no microcode `INTERR`); on an indirect
call it types that site. Empty/`NULL` clears the override.

---

## blocks
Basic blocks within functions. **Use `func_addr` constraint for performance.**

| Column | Type | Description |
|--------|------|-------------|
| `func_addr` | INT | Containing function |
| `start_addr` | INT | Block start |
| `end_addr` | INT | Block end |
| `size` | INT | Block size |

```sql
-- Blocks in a specific function (FAST - uses constraint pushdown)
SELECT * FROM blocks WHERE func_addr = 0x401000;

-- Functions with most basic blocks
SELECT (SELECT name FROM funcs WHERE func_addr >= addr AND func_addr < end_addr LIMIT 1) as name, COUNT(*) as blocks
FROM blocks GROUP BY func_addr ORDER BY blocks DESC LIMIT 10;
```

### cfg_edges

Control flow graph edges between basic blocks. **Always use `WHERE func_addr = X`** (filter_eq pushdown, O(blocks in function)).

| Column | Type | Description |
|--------|------|-------------|
| `func_addr` | INT | Containing function |
| `from_block` | INT | Source block address |
| `to_block` | INT | Target block address |
| `edge_type` | TEXT | `normal` (single-successor or fallback label), `true`/`false` (generic first/second arms for a two-way branch; labels follow successor order, not taken/fallthrough semantics) |

```sql
-- Get CFG structure
SELECT * FROM cfg_edges WHERE func_addr = 0x401000;

-- Find branch points (conditional blocks)
SELECT from_block, COUNT(*) as succ_count
FROM cfg_edges WHERE func_addr = 0x401000
GROUP BY from_block HAVING succ_count > 1;

-- Find merge points (blocks with multiple predecessors)
SELECT to_block, COUNT(*) as pred_count
FROM cfg_edges WHERE func_addr = 0x401000
GROUP BY to_block HAVING pred_count > 1;

-- Function complexity ranking: combine CFG, loops, and call metrics
SELECT f.name, f.size,
       (SELECT COUNT(*)
        FROM (
            SELECT ce.from_block
            FROM cfg_edges ce
            WHERE ce.func_addr = f.addr
            GROUP BY ce.from_block
            HAVING COUNT(*) > 1
        ) branch_blocks) as branch_sites,
       (SELECT COUNT(*) FROM disasm_loops dl WHERE dl.func_addr = f.addr) as loops,
       (SELECT COUNT(*) FROM disasm_calls dc WHERE dc.func_addr = f.addr) as calls_made
FROM funcs f
WHERE f.size > 32
ORDER BY branch_sites DESC
LIMIT 20;
```

### function_chunks

Cached table with one row per function chunk. Aggregate by `func_addr` when you
need function-level span or density metrics.

| Column | Type | Description |
|--------|------|-------------|
| `func_addr` | INT | Function address |
| `chunk_start` | INT | Chunk start address |
| `chunk_end` | INT | Chunk end address |
| `block_count` | INT | Number of blocks in chunk |
| `total_size` | INT | Total size of chunk |

```sql
SELECT * FROM function_chunks WHERE func_addr = 0x401000;
```

---

## SQL Functions -- Disassembly

| Function | Description |
|----------|-------------|
| `disasm_at(addr)` | Canonical listing line for containing head (works for code/data) |
| `disasm_at(addr, n)` | Canonical listing line with +/- `n` neighboring heads |
| `disasm(addr)` | Single disassembly line at address |
| `disasm(addr, n)` | Next N instructions from address (count-based) |
| `disasm_range(start, end)` | All disassembly lines in address range [start, end) |
| `disasm_func(addr)` | Full disassembly of function containing address |
| `make_code(addr)` | Create instruction at address (returns 1/0) |
| `make_code_range(start, end)` | Create instructions in range, returns created count |

### Disassembly Examples

```sql
-- Canonical single-EA disassembly (safe for code or data)
SELECT disasm_at(0x401000);

-- Canonical context window (+/- 2 heads)
SELECT disasm_at(0x401000, 2);

-- Full function disassembly (resolves boundaries via get_func)
SELECT disasm_func(addr) FROM funcs WHERE name = '_main';

-- Disassemble a specific address range
SELECT disasm_range(0x401000, 0x401100);

-- Sliding window: next 5 instructions from an address
SELECT disasm(0x401000, 5);
```

---

## SQL Functions -- Names & Functions

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

## SQL Functions -- Navigation

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

## SQL Functions -- Item Analysis

Use `heads` for item classification, size, and raw flags:

```sql
SELECT addr, size, type, flags, disasm
FROM heads
WHERE addr = 0x401000;
```

---

## SQL Functions -- Instruction Details

Use `instructions` and `instruction_operands` for decoded instruction facts:

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

## SQL Functions -- File Generation

| Function | Description |
|----------|-------------|
| `gen_listing(path)` | Generate full-database listing output (LST) |

---

## SQL Functions -- Graph Generation

| Function | Description |
|----------|-------------|
| `gen_cfg_dot(addr)` | Generate CFG as DOT graph string |
| `gen_cfg_dot_file(addr, path)` | Write CFG DOT to file |
| `gen_schema_dot()` | Generate database schema as DOT |

```sql
-- Get CFG for a function as DOT format
SELECT gen_cfg_dot(0x401000);
```

---

## Performance Rules

| Table | Architecture | Key Constraint | Notes |
|-------|-------------|----------------|-------|
| `funcs` | Index-Based | none needed | O(1) per row via `getn_func(i)` -- always fast |
| `instructions` | Iterator | `func_addr` | Function-item iterator (fast) vs full code-head scan (slow) |
| `blocks` | Iterator | `func_addr` | Constraint pushdown: iterates blocks of one function |
| `cfg_edges` | Iterator | `func_addr` | filter_eq pushdown: O(blocks in function) |
| `disasm_calls` | Generator | `func_addr`, `addr` for writes | Lazy streaming, respects LIMIT; `callee_type` is writable |
| `heads` | Generator | `addr =`, addr range | Consumes `ORDER BY addr` for next/previous navigation; broad scans can still be large |
| `instruction_operands` | Iterator | `addr`, `func_addr` | Address lookup decodes one instruction; function lookup iterates one function |
| `segments` | Index-Based | none needed | Small table, always fast |
| `names` | Iterator | none needed | Iterates IDA's name list |

**Key rules:**
- `funcs` is always fast -- no constraint needed.
- `instructions` without `func_addr` scans every code head -- use `func_addr` for per-function queries.
- `blocks` without `func_addr` iterates all functions' flowcharts -- always constrain.
- `heads` is often large. Use `addr = X` for item facts and address range plus `ORDER BY addr [DESC] LIMIT 1` for navigation.

**Cost model:**
```
funcs (full scan)            -> O(number of functions), typically ~1000s, fast
instructions WHERE func_addr -> O(function_size / avg_insn_size)
instructions (no constraint) -> O(total_code_heads), potentially 100K+
blocks WHERE func_addr         -> O(block_count_in_func), fast
cfg_edges WHERE func_addr      -> O(block_count_in_func), fast
disasm_calls WHERE func_addr -> O(instructions_in_func), streaming
disasm_calls UPDATE callee_type WHERE addr -> O(1) call-site lookup
heads WHERE addr          -> O(1) IDA head check
heads next/prev LIMIT 1      -> O(distance to next/previous defined head)
instruction_operands addr    -> O(operands in one instruction)
instruction_operands func    -> O(operands in one function)
```

---

## Additional Resources

- For advanced CTE patterns and instruction lifecycle playbooks: [references/disassembly-examples.md](references/disassembly-examples.md)
- For additional table schemas (fchunks, heads, bytes, signatures, problems, fixups, etc.): [references/disassembly-tables.md](references/disassembly-tables.md)

---

## See Also

- `decompiler` — pseudocode and ctree on the same function; pivot when disassembly is too noisy.
- `xrefs` — call graph and data references centered on a function or call site.
- `data` — operand-targeted bytes/strings; `bytes` table for per-byte reads and bounded windows via `WHERE start_addr = X AND n = N` composed with `hex(blob_concat(value))` for bulk hex.

