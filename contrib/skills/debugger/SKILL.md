---
name: debugger
description: "Use for IDA debugger ops: breakpoints, patches, conditions, patch inventory."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: engineering
  upstream: C:\Users\Admin\.agents\skills\debugger\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\debugger\SKILL.md

## Trigger Intents

Use this skill when user asks to:
- add/remove/modify breakpoints
- patch bytes or revert patches
- create patch inventories and debugging action plans
- instrument analysis-driven break/watch workflows

Route to:
- `analysis`/`xrefs` for selecting meaningful targets first
- `disassembly` for opcode-level patch context
- `annotations` for documenting patch rationale and outcomes

---

## Do This First (Warm-Start Sequence)

```sql
-- 1) Current breakpoint inventory
SELECT printf('0x%X', addr) AS addr, type_name, enabled
FROM breakpoints
ORDER BY addr;

-- 2) Current patch inventory (fast: backed by IDA's patch list)
SELECT printf('0x%X', addr) AS addr, original_value, value AS patched_value
FROM bytes WHERE is_patched = 1
ORDER BY addr
LIMIT 50;

-- 3) Validate target bytes before patch
SELECT addr, value, original_value, is_patched
FROM bytes
WHERE addr = 0x401000;
```

Interpretation guidance:
- Confirm existing instrumentation before adding more.
- Always snapshot current/original byte state before mutating.

---

## Failure and Recovery

- Breakpoint insert/update failed:
  - Validate address existence and hardware size/type compatibility.
- Patch verification mismatch:
  - Re-read `bytes` (and `WHERE is_patched = 1`), then retry with precise address.
- Unintended patch side effects:
  - Revert with `DELETE FROM bytes WHERE addr = ...` and reassess target instruction context.

---

## Handoff Patterns

1. `debugger` -> `disassembly` to validate instruction semantics around patch site.
2. `debugger` -> `xrefs` to assess blast radius of patched/broken call paths.
3. `debugger` -> `annotations` to leave durable analyst breadcrumbs.

---

## breakpoints

Debugger breakpoints. Supports full CRUD (SELECT, INSERT, UPDATE, DELETE). Breakpoints persist in the IDB even without an active debugger session. `folder_path` is the folder-oriented spelling of IDA's breakpoint `group`; updating either updates the same underlying breakpoint grouping.

| Column | Type | RW | Description |
|--------|------|----|-------------|
| `addr` | INT | R | Breakpoint address |
| `enabled` | INT | RW | 1=enabled, 0=disabled |
| `type` | INT | RW | Breakpoint type (1=hw_write, 2=hw_read, 3=hw_rdwr, 4=software, 8=hw_exec) |
| `type_name` | TEXT | R | Type name (software, hardware_write, etc.) |
| `size` | INT | RW | Breakpoint size (for hardware breakpoints) |
| `flags` | INT | RW | Breakpoint flags |
| `pass_count` | INT | RW | Pass count before trigger |
| `condition` | TEXT | RW | Condition expression |
| `loc_type` | INT | R | Location type code |
| `loc_type_name` | TEXT | R | Location type (absolute, relative, symbolic, source) |
| `module` | TEXT | R | Module path (relative breakpoints) |
| `symbol` | TEXT | R | Symbol name (symbolic breakpoints) |
| `offset` | INT | R | Offset (relative/symbolic) |
| `source_file` | TEXT | R | Source file (source breakpoints) |
| `source_line` | INT | R | Source line number |
| `is_hardware` | INT | R | 1=hardware breakpoint |
| `is_active` | INT | R | 1=currently active |
| `group` | TEXT | RW | Breakpoint group name |
| `bptid` | INT | R | Breakpoint ID |
| `folder_path` | TEXT | RW | Alias for breakpoint group folder; `NULL` means root |
| `full_path` | TEXT | R | Full breakpoint dirtree path when available |

```sql
-- List all breakpoints
SELECT printf('0x%08X', addr) as addr, type_name, enabled, condition
FROM breakpoints;

-- Add software breakpoint
INSERT INTO breakpoints (addr) VALUES (0x401000);

-- Add hardware write watchpoint
INSERT INTO breakpoints (addr, type, size) VALUES (0x402000, 1, 4);

-- Add conditional breakpoint (at a different address than the plain one above)
INSERT INTO breakpoints (addr, condition) VALUES (0x401330, 'eax == 0');

-- Move breakpoint into/out of an IDA breakpoint folder
UPDATE breakpoints SET folder_path = 'idasql/breakpoints/network' WHERE addr = 0x401000;
UPDATE breakpoints SET folder_path = NULL WHERE addr = 0x401000;

-- Disable a breakpoint
UPDATE breakpoints SET enabled = 0 WHERE addr = 0x401000;

-- Delete a breakpoint
DELETE FROM breakpoints WHERE addr = 0x401000;

-- Find which functions have breakpoints
SELECT b.addr, f.name, b.type_name, b.enabled
FROM breakpoints b
JOIN funcs f ON b.addr >= f.addr AND b.addr < f.end_addr;
```

---

## bytes (Byte Patching)

Pure mapped-byte program view with patch support. This table is one row per
mapped byte address; IDA item metadata such as size/type belongs to `heads`.

| Column | Type | RW | Description |
|--------|------|----|-------------|
| `addr` | INT | R | Byte address |
| `value` | INT | RW | Current byte value (UPDATE patches 1 byte) |
| `word` | INT | RW | 2-byte little-endian value (UPDATE patches 2 bytes) |
| `dword` | INT | RW | 4-byte little-endian value (UPDATE patches 4 bytes) |
| `qword` | INT | RW | 8-byte little-endian value (UPDATE patches 8 bytes) |
| `original_value` | INT | R | Original byte value before patch |
| `is_patched` | INT | R | 1 if byte differs from original (`WHERE is_patched = 1` enumerates patches fast) |
| `fpos` | INT | R | Physical/input file offset (NULL when unavailable) |

Revert a patch with `DELETE FROM bytes WHERE addr = ...` (or `WHERE is_patched = 1`).

```sql
-- Read one address
SELECT addr, value, original_value, is_patched
FROM bytes WHERE addr = 0x401000;

-- Read a byte range, including item-tail bytes
SELECT addr, value
FROM bytes
WHERE addr >= 0x401000 AND addr < 0x401010
ORDER BY addr;

-- Get item metadata separately
SELECT addr, size, type, flags, disasm
FROM heads
WHERE addr = 0x401000;

-- Patch via table update (single byte, or width columns word/dword/qword)
UPDATE bytes SET value = 0x90 WHERE addr = 0x401000;
UPDATE bytes SET dword = 0x90909090 WHERE addr = 0x401000;  -- little-endian

-- Inspect patch inventory (fast: backed by IDA's patch list)
SELECT printf('0x%X', addr) AS addr, original_value, value AS patched_value
FROM bytes WHERE is_patched = 1 ORDER BY addr LIMIT 20;

-- Persist once done
SELECT save_database();
```

---

## bytes — Patching (write surface)

All byte patching is done through the writable `bytes` table.

| Column | RW | Meaning |
|--------|----|---------|
| `value` | RW | Current byte; UPDATE patches 1 byte |
| `word` / `dword` / `qword` | RW | UPDATE patches 2/4/8 bytes little-endian |
| `original_value` | R | Original (pre-patch) byte |
| `is_patched` | R | 1 if patched; `WHERE is_patched = 1` enumerates patches fast |

```sql
SELECT printf('0x%X', addr) AS addr,
       printf('0x%02X', original_value) AS old,
       printf('0x%02X', value) AS new
FROM bytes WHERE is_patched = 1
ORDER BY addr;
```

---

## Byte Access via the `bytes` Table

All byte reads and patches go through the `bytes` table. Bounded-read
shapes are documented in `data`; this skill focuses on the patching
workflow. `load_file_bytes(...)` writes a file's bytes into the IDB at
a given range; use it when the patch content already lives in a file.

| Function / Shape | Description |
|------------------|-------------|
| `SELECT hex(blob_concat(value)) FROM (SELECT value FROM bytes WHERE start_addr = X AND n = N ORDER BY addr)` | Read N bytes as uppercase hex |
| `SELECT blob_concat(value) FROM (SELECT value FROM bytes WHERE start_addr = X AND n = N ORDER BY addr)` | Read N bytes as BLOB |
| `load_file_bytes(path, file_offset, addr, size[, patchable])` | Write a file's bytes into the IDB at the target range |

```sql
-- Read 16 bytes as hex
SELECT hex(blob_concat(value))
FROM (SELECT value FROM bytes WHERE start_addr = 0x401000 AND n = 16 ORDER BY addr);

-- Patch one byte (example: NOP) and a 4-byte little-endian value
UPDATE bytes SET value = 0x90 WHERE addr = 0x401000;
UPDATE bytes SET dword = 0x90909090 WHERE addr = 0x401000;

-- Verify current vs original
SELECT value AS current, original_value AS original
FROM bytes WHERE addr = 0x401000;

-- Revert patch (one byte, or every patch)
DELETE FROM bytes WHERE addr = 0x401000;
DELETE FROM bytes WHERE is_patched = 1;

-- Persist patches explicitly
SELECT save_database();
```

---

## Analysis-Driven Breakpoint Workflows

### Set breakpoints on all callers of a security-sensitive API

Use disasm_calls to find every call site and batch-insert breakpoints:

```sql
-- Breakpoint on every call to VirtualAlloc (or similar)
INSERT INTO breakpoints (addr)
SELECT addr FROM disasm_calls WHERE callee_name LIKE '%VirtualAlloc%';

-- Verify
SELECT printf('0x%08X', addr) AS addr, type_name, enabled
FROM breakpoints;
```

### Watchpoints on struct fields discovered via type analysis

After recovering a struct, set hardware watchpoints on specific field offsets:

```sql
-- Hardware write watchpoint on a 4-byte field (e.g., config.flags at base+0x10)
-- First, find where the struct base is stored (requires manual analysis)
INSERT INTO breakpoints (addr, type, size) VALUES (0x402010, 1, 4);
-- type=1 is hardware_write, size=4 for DWORD field
```

### Conditional breakpoints from decompiler analysis

Set breakpoints that only trigger when specific conditions are met:

```sql
-- Break when first argument (rcx on x64 fastcall) equals a specific enum value
INSERT INTO breakpoints (addr, condition)
VALUES (0x401000, 'rcx == 3');

-- Break on error return
INSERT INTO breakpoints (addr, condition)
VALUES (0x401050, 'rax == 0xFFFFFFFF');
```

---

## Patching Workflows

### NOP out anti-debug checks

Find and neutralize `IsDebuggerPresent` checks:

```sql
-- Find calls to IsDebuggerPresent
SELECT dc.addr, (SELECT name FROM funcs WHERE dc.func_addr >= addr AND dc.func_addr < end_addr LIMIT 1) AS func_name,
       disasm_at(dc.addr, 2) AS context
FROM disasm_calls dc
WHERE dc.callee_name LIKE '%IsDebuggerPresent%';

-- Patch the conditional jump after the check (example: jnz → nop nop)
-- First inspect the instruction after the call
SELECT disasm_at(0x401030, 3);
-- Then patch (adjust addresses based on actual binary)
UPDATE bytes SET value = 0x90 WHERE addr = 0x401035;
UPDATE bytes SET value = 0x90 WHERE addr = 0x401036;
```

### Inventory all patches and generate report

```sql
-- Full patch report: what was changed and where
SELECT printf('0x%X', addr) AS addr,
       (SELECT name FROM funcs WHERE addr >= addr AND addr < end_addr LIMIT 1) AS func_name,
       printf('0x%02X', original_value) AS original,
       printf('0x%02X', value) AS patched,
       disasm_at(addr) AS context
FROM bytes WHERE is_patched = 1
ORDER BY addr;
```

---

## Performance Notes

| Table | Size | Constraint | Notes |
|-------|------|-----------|-------|
| `breakpoints` | Small (<100 typical) | none needed | Always fast |
| `bytes` | All mapped bytes | `addr` | **Critical** — constrain to one address or a tight range |
| `bytes WHERE is_patched = 1` | Small (patch count) | `is_patched = 1` | Fast — iterates only patched locations via IDA's patch list |

- `breakpoints` table is small — full scans are fine.
- `bytes` table emits one row per mapped byte. Use `WHERE addr = X` or a tight `addr` range.
- `bytes WHERE is_patched = 1` iterates only patched locations — always fast.

---

## See Also

- `disassembly` — instruction context around a patch site (`disasm_at`, operand semantics).
- `annotations` — record patch rationale via comments/bookmarks before mutating.
- `data` — canonical owner of byte-read shapes (table vs. scalar disambiguation).

