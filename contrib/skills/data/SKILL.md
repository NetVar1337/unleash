---
name: data
description: "Use to query IDA strings/bytes/binary content and rebuild string tables."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: imported
  upstream: C:\Users\Admin\.agents\skills\data\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\data\SKILL.md

## Trigger Intents

Use this skill when user asks to:
- search strings/bytes/patterns quickly
- map string evidence to code usage
- investigate raw data-level indicators (IOCs, constants, signatures)

Route to:
- `xrefs` for relationship expansion from matched addresses
- `analysis` for triage synthesis from data signals
- `debugger` when findings should drive patch/breakpoint actions

---

## Do This First (Warm-Start Sequence)

```sql
-- 1) Validate string availability
SELECT COUNT(*) AS strings FROM strings;

-- 2) Sample high-value strings
SELECT content, printf('0x%X', addr) AS addr
FROM strings
WHERE length >= 8
ORDER BY length DESC
LIMIT 40;

-- 3) If expected strings are missing, rebuild once
SELECT rebuild_strings();
```

Interpretation guidance:
- Strings are often quickest behavioral clues; pivot to `xrefs` immediately for execution context.
- For opcode/pattern hunts, prefer the `byte_search` table over full instruction scans.

---

## Failure and Recovery

- No strings or unexpectedly low count:
  - Run `rebuild_strings()` and validate with `COUNT(*) FROM strings`.
- Too many false positives:
  - Increase specificity (`LIKE`, regex-like pattern narrowing, module/function join filters).
- Byte pattern search too broad:
  - Restrict by range or join matched byte addresses to `funcs`.
- Need named functions, labels, structs, or members instead of string contents:
  - Use `grep`, not `strings` or `byte_search`.

---

## Handoff Patterns

1. `data` -> `xrefs` to convert data hits into code paths/functions.
2. `data` -> `analysis` for risk scoring and campaign-level insight.
3. `data` -> `debugger` for actionable patch/watchpoint setup.

---

## strings
String literals found in the binary. IDA maintains a cached string list that can be configured.

| Column | Type | Description |
|--------|------|-------------|
| `addr` | INT | String address |
| `length` | INT | String length |
| `type` | INT | String type (raw encoding bits) |
| `type_name` | TEXT | Type name: ascii, utf16, utf32 |
| `width` | INT | Char width (0=1-byte, 1=2-byte, 2=4-byte) |
| `width_name` | TEXT | Width name: 1-byte, 2-byte, 4-byte |
| `layout` | INT | String layout (0=null-terminated, 1-3=pascal) |
| `layout_name` | TEXT | Layout name: termchr, pascal1, pascal2, pascal4 |
| `encoding` | INT | Encoding index (0=default) |
| `content` | TEXT | String content (the actual text — not `value` or `text`) |

**String Type Encoding:**
IDA stores string type as a 32-bit value:
- Bits 0-1: Width (0=1B/ASCII, 1=2B/UTF-16, 2=4B/UTF-32)
- Bits 2-7: Layout (0=TERMCHR, 1=PASCAL1, 2=PASCAL2, 3=PASCAL4)
- Bits 8-15: term1 (first termination character)
- Bits 16-23: term2 (second termination character)
- Bits 24-31: encoding index

```sql
-- Find error messages
SELECT content, printf('0x%X', addr) as addr FROM strings WHERE content LIKE '%error%';

-- ASCII strings only
SELECT * FROM strings WHERE type_name = 'ascii';

-- UTF-16 strings (common in Windows)
SELECT * FROM strings WHERE type_name = 'utf16';

-- Count strings by type
SELECT type_name, layout_name, COUNT(*) as count
FROM strings GROUP BY type_name, layout_name ORDER BY count DESC;
```

**Important:** For new analysis (exe/dll), strings are auto-built. For existing databases (i64/idb), strings are already saved. If you see 0 strings unexpectedly, run `SELECT rebuild_strings()` once to rebuild the list. See String List Surfaces section below.

---

## String References (explicit join pattern)

Use `strings + xrefs + funcs` directly. This is the canonical pattern.

```sql
-- Find call sites/functions referencing error-like strings
SELECT
    s.content as string_value,
    printf('0x%X', x.from_addr) as ref_addr,
    (SELECT name FROM funcs WHERE x.from_addr >= addr AND x.from_addr < end_addr LIMIT 1) as func_name
FROM strings s
JOIN xrefs x ON x.to_addr = s.addr
WHERE s.content LIKE '%error%' OR s.content LIKE '%fail%'
ORDER BY func_name, ref_addr;

-- Functions with most string references
SELECT
    (SELECT name FROM funcs WHERE x.from_addr >= addr AND x.from_addr < end_addr LIMIT 1) as func_name,
    COUNT(*) as string_refs
FROM strings s
JOIN xrefs x ON x.to_addr = s.addr
GROUP BY func_name
ORDER BY string_refs DESC
LIMIT 10;
```

---

## `bytes` Table — Reads, Patches, and Bounded Windows

**All byte access goes through the `bytes` table.** The hidden `start_addr` and
`n` input columns pair up for bounded reads; bulk hex uses
`hex(blob_concat(value))`, bulk BLOB uses `blob_concat(value)`.

| Shape | SQL |
|-------|-----|
| Read 1 byte | `SELECT value FROM bytes WHERE addr = 0x401000` |
| Read N bytes as hex (uppercase, no spaces) | `SELECT hex(blob_concat(value)) FROM (SELECT value FROM bytes WHERE start_addr = 0x401000 AND n = 16 ORDER BY addr)` |
| Read N bytes as BLOB | `SELECT blob_concat(value) FROM (SELECT value FROM bytes WHERE start_addr = 0x401000 AND n = 16 ORDER BY addr)` |
| Read a range | `SELECT value FROM bytes WHERE addr >= 0x401000 AND addr < 0x401010 ORDER BY addr` |

The hidden `start_addr` + `n` columns request exactly N consecutive bytes
beginning at X. They are deliberately distinct from the visible `addr` column
so any predicate on `addr` (joins, compound `WHERE`) stays enforceable by
SQLite. The bounded read does not skip unmapped addresses; rows beyond the
mapped region report whatever `get_byte()` yields there.

`blob_concat(value)` is a built-in aggregate that concatenates row values
into one BLOB; `hex()` is the SQLite built-in BLOB→hex helper (uppercase).

### Unbounded-range gotcha

`WHERE addr > X` **without an upper bound or LIMIT** walks every mapped byte
from X to end-of-image — millions of rows and seconds of wall time. Always
pair the read with one of:

- `WHERE start_addr = X AND n = N` (bounded read shape)
- `AND addr < B` (two-sided range)
- outer `LIMIT N`

The table has no per-call cap; arbitrarily large windows are supported as
long as the constraint pair is bounded.

Use `heads` when you need IDA item size/type metadata.

---

## Binary Search Table

Use `byte_search` for raw bytes/opcodes. It requires `WHERE pattern = ...`; `matched_hex` is an output column, not the search input.

| Column | Type | Description |
|--------|------|-------------|
| `addr` | INT | Match address |
| `matched_hex` | TEXT | Matched bytes rendered as hex text |
| `matched_bytes` | BLOB | Matched bytes as raw bytes |
| `size` | INT | Match size in bytes |
| `pattern` | HIDDEN TEXT | Required IDA byte pattern input |
| `start_addr` | HIDDEN INT | Optional inclusive lower bound |
| `end_addr` | HIDDEN INT | Optional exclusive upper bound |
| `max_results` | HIDDEN INT | Optional generator cap |

**Pattern syntax (IDA native):**
- `"48 8B 05"` - Exact bytes (hex, space-separated)
- `"48 ? 05"` or `"48 ?? 05"` - `?` = any byte wildcard (whole byte only)
- `"(01 02 03)"` - Alternatives (match any of these bytes)

**Note:** Nibble wildcards and regex are not supported in byte patterns.

**Example:**
```sql
-- Find all matches for a pattern
SELECT addr, matched_hex, size
FROM byte_search
WHERE pattern = '48 8B ? 00'
LIMIT 10;

-- First match only
SELECT printf('0x%llX', addr) AS addr
FROM byte_search
WHERE pattern = 'CC CC CC'
ORDER BY addr
LIMIT 1;

-- Search with alternatives
SELECT addr, matched_hex
FROM byte_search
WHERE pattern = 'E8 (01 02 03 04)'
LIMIT 20;
```

**Optimization Pattern: Find functions using specific instruction**

To answer "How many functions use RDTSC instruction?" efficiently:
```sql
-- Count unique functions containing RDTSC (opcode: 0F 31)
SELECT COUNT(DISTINCT f.addr) as count
FROM byte_search b
JOIN funcs f ON b.addr >= f.addr AND b.addr < f.end_addr
WHERE b.pattern = '0F 31';

-- List those functions with names
SELECT DISTINCT
    f.addr as func_addr,
    f.name as func_name
FROM byte_search b
JOIN funcs f ON b.addr >= f.addr AND b.addr < f.end_addr
WHERE b.pattern = '0F 31';
```

This is **much faster** than scanning all disassembly lines because:
- `byte_search` uses IDA's native binary search
- the containment join uses the compact `funcs` table instead of scanning every instruction

---

## Choose the Right Search Surface

- Use `grep` for named entities such as functions, labels, structs, enums, and members.
- Use `strings` when the user is searching literal string contents inside the binary.
- Use `byte_search` when the target is a raw byte or opcode pattern.

---

## SQL Surfaces — String List

IDA maintains a cached list of strings. Use `rebuild_strings()` to detect and cache strings, `COUNT(*) FROM strings` for the current count, and `strings` for row-level analysis.

| Surface | Description |
|---------|-------------|
| `rebuild_strings()` | Rebuild with ASCII + UTF-16, minlen 5 (default) |
| `rebuild_strings(minlen)` | Rebuild with custom minimum length |
| `rebuild_strings(minlen, types)` | Rebuild with custom length and type mask |
| `SELECT COUNT(*) FROM strings` | Current string-list count (optimized without row materialization) |

**Type mask values:**
- `1` = ASCII only (STRTYPE_C)
- `2` = UTF-16 only (STRTYPE_C_16)
- `4` = UTF-32 only (STRTYPE_C_32)
- `3` = ASCII + UTF-16 (default)
- `7` = All types

```sql
-- Check current string count
SELECT COUNT(*) AS strings FROM strings;

-- Rebuild with defaults (ASCII + UTF-16, minlen 5)
SELECT rebuild_strings();

-- Rebuild with shorter minimum length
SELECT rebuild_strings(4);

-- Rebuild with specific types
SELECT rebuild_strings(5, 1);   -- ASCII only
SELECT rebuild_strings(5, 7);   -- All types (ASCII + UTF-16 + UTF-32)

-- Typical workflow: rebuild then query
SELECT rebuild_strings();
SELECT * FROM strings WHERE content LIKE '%error%';
```

**IMPORTANT - Agent Behavior for String Queries:**
When the user asks about strings (e.g., "show me the strings", "what strings are in this binary"):
1. First run `SELECT rebuild_strings()` to ensure strings are detected
2. Then query the `strings` table

The `rebuild_strings()` function configures IDA's string detection with sensible defaults (ASCII + UTF-16, minimum length 5) and rebuilds the string list. This ensures the user gets results even if the database had no prior string analysis.

---

## Performance Rules

| Table/Function | Architecture | Notes |
|----------------|-------------|-------|
| `COUNT(*) FROM strings` | Cached table count path | O(1) current string-list count |
| `strings` | Query-scoped | Materialized from IDA's live string list on each scan (no persisted vtable cache); fast because IDA's `strlist` is already built. `rebuild_strings()` regenerates IDA's underlying string list |
| `byte_search` | Native binary search table | Much faster than iterating instructions table |
| `bytes WHERE addr = X` | Point lookup | O(1); virtual table index |
| `bytes WHERE start_addr = X AND n = N` | Bounded read via hidden `start_addr` + `n` | O(N); virtual table index |
| `bytes WHERE addr >= A AND addr < B` | Range scan | O(range); virtual table index |
| `bytes WHERE addr > X` (unbounded) | Range scan | **AVOID** — walks every mapped byte to end-of-image; use bounded forms |

**Key rules:**
- Always call `rebuild_strings()` before the first string query on a new database or after making code/data changes that may create new strings.
- `byte_search` uses IDA's native binary search engine; for "find functions containing opcode X", join matches to `funcs` by containment instead of scanning the `instructions` table.
- `strings` table is a snapshot of the cached string list. If IDA's analysis creates new strings after your initial query, call `rebuild_strings()` again.
- For cross-referencing strings with functions, the `strings + xrefs + funcs` JOIN pattern is canonical — IDA's xref index makes the JOIN fast.

---

## Advanced Data Patterns (CTEs)

### Security-relevant string triage

Categorize strings by security relevance for rapid threat assessment:

```sql
-- Categorize strings by security relevance
SELECT rebuild_strings();

WITH categorized AS (
    SELECT addr, content,
        CASE
            WHEN content LIKE '%password%' OR content LIKE '%passwd%' OR content LIKE '%secret%'
                THEN 'credential'
            WHEN content LIKE '%http://%' OR content LIKE '%https://%' OR content LIKE '%ftp://%'
                THEN 'url'
            WHEN content LIKE '%error%' OR content LIKE '%fail%' OR content LIKE '%exception%'
                THEN 'error'
            WHEN content LIKE '%debug%' OR content LIKE '%trace%' OR content LIKE '%assert%'
                THEN 'debug'
            WHEN content LIKE '%.exe%' OR content LIKE '%.dll%' OR content LIKE '%.sys%'
                THEN 'file_path'
            WHEN content LIKE '%HKEY_%' OR content LIKE '%SOFTWARE\\%'
                THEN 'registry'
            ELSE 'other'
        END AS category
    FROM strings
    WHERE length >= 5
)
SELECT category, COUNT(*) AS count,
       GROUP_CONCAT(SUBSTR(content, 1, 60), ' | ') AS samples
FROM categorized
WHERE category != 'other'
GROUP BY category
ORDER BY count DESC;
```

### Combine string references with function size for suspicion scoring

Functions referencing security-relevant strings AND having significant size are high-priority targets:

```sql
-- Suspicious functions: reference interesting strings AND are non-trivial
WITH interesting_strings AS (
    SELECT addr, content FROM strings
    WHERE content LIKE '%password%' OR content LIKE '%encrypt%'
       OR content LIKE '%decrypt%' OR content LIKE '%http%'
       OR content LIKE '%socket%' OR content LIKE '%connect%'
),
string_funcs AS (
    SELECT DISTINCT (SELECT addr FROM funcs WHERE x.from_addr >= addr AND x.from_addr < end_addr LIMIT 1) AS func_addr,
           s.content AS matched_string
    FROM interesting_strings s
    JOIN xrefs x ON x.to_addr = s.addr
    WHERE (SELECT addr FROM funcs WHERE x.from_addr >= addr AND x.from_addr < end_addr LIMIT 1) IS NOT NULL
)
SELECT (SELECT name FROM funcs WHERE sf.func_addr >= addr AND sf.func_addr < end_addr LIMIT 1) AS func_name,
       printf('0x%X', sf.func_addr) AS addr,
       f.size AS func_size,
       GROUP_CONCAT(sf.matched_string, ' | ') AS strings_referenced
FROM string_funcs sf
JOIN funcs f ON f.addr = sf.func_addr
GROUP BY sf.func_addr
ORDER BY f.size DESC
LIMIT 20;
```

### Find functions referencing both crypto-related and network-related strings

Cross-category correlation for identifying data exfiltration or C2 communication:

```sql
-- Functions touching both crypto and network strings
WITH crypto_refs AS (
    SELECT DISTINCT (SELECT addr FROM funcs WHERE x.from_addr >= addr AND x.from_addr < end_addr LIMIT 1) AS func_addr
    FROM strings s JOIN xrefs x ON x.to_addr = s.addr
    WHERE s.content LIKE '%crypt%' OR s.content LIKE '%aes%'
       OR s.content LIKE '%cipher%' OR s.content LIKE '%hash%'
),
network_refs AS (
    SELECT DISTINCT (SELECT addr FROM funcs WHERE x.from_addr >= addr AND x.from_addr < end_addr LIMIT 1) AS func_addr
    FROM strings s JOIN xrefs x ON x.to_addr = s.addr
    WHERE s.content LIKE '%socket%' OR s.content LIKE '%connect%'
       OR s.content LIKE '%send%' OR s.content LIKE '%recv%'
       OR s.content LIKE '%http%'
)
SELECT (SELECT name FROM funcs WHERE c.func_addr >= addr AND c.func_addr < end_addr LIMIT 1) AS func_name,
       printf('0x%X', c.func_addr) AS addr
FROM crypto_refs c
JOIN network_refs n ON n.func_addr = c.func_addr;
```

---

## See Also

- `disassembly` — where strings/bytes are referenced in code (operand-targeted reads).
- `xrefs` — data references to a target address; canonical pivot from a data hit to its consumers.
- `debugger` — byte patching uses the `bytes` table; this skill owns the *read* shapes.
- `analysis` — strings/imports as triage signals.

