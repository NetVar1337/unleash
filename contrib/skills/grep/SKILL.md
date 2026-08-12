---
name: grep
description: "Use to find IDA entities by name pattern and seed xref/decompiler workflows."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: imported
  upstream: C:\Users\Admin\.agents\skills\grep\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\grep\SKILL.md

`grep` is IDASQL's entity-search surface. Use it to discover named functions, labels, segments, structs, enums, and members before pivoting into xrefs, decompiler, or type work.

---

## Trigger Intents

Use this skill when user asks to:
- find functions, labels, types, or members by name
- search by prefix/substring like `sub_`, `EH`, `Zw`, `CreateFile`, or `main`
- page through search results quickly
- seed table-native xref/decompiler/type workflows from name discovery

Route to:
- `xrefs` after locating a candidate callee/import/function and needing callers/callees/references
- `decompiler` after choosing a candidate function to inspect semantically
- `types` when the hit is a struct/enum/member you need to inspect or edit

---

## Do This First (Quick Start)

```sql
-- 1) Start with a structured search while you learn the result shape
SELECT name, kind, addr
FROM grep
WHERE pattern = 'main'
ORDER BY kind, name
LIMIT 20;
```

```sql
-- 2) Narrow immediately when the result set is noisy
SELECT name, ordinal, full_name
FROM grep
WHERE pattern = 'EH%' AND kind = 'struct'
ORDER BY name;
```

```sql
-- 3) Page with ordinary SQL
SELECT name, kind, addr
FROM grep
WHERE pattern = 'sub_%'
ORDER BY kind, name
LIMIT 10 OFFSET 10;
```

Interpretation guidance:
- `grep` is a table. Use normal SQL for filtering, sorting, joining, grouping, and paging.
- For downstream parsing, select rows directly instead of wrapping the results in JSON.

---

## Result Shape

`grep` exposes named IDA entities as rows:
- `name`
- `kind`
- `addr`
- `ordinal`
- `parent_name`
- `full_name`

Common `kind` values:
- `function`
- `label`
- `segment`
- `struct`
- `union`
- `enum`
- `member`
- `enum_member`

---

## Pattern Rules

- Matching is case-insensitive.
- Plain text becomes a contains-match.
- `%` matches any substring.
- `_` matches a single character.
- `*` is accepted and normalized to `%`.
- Empty pattern returns no rows.
- This is not regex.
- This is unrelated to `byte_search`.

Examples:

```sql
-- Contains-match
SELECT name, kind
FROM grep
WHERE pattern = 'main'
LIMIT 20;
```

```sql
-- Prefix wildcard
SELECT name, kind, addr
FROM grep
WHERE pattern = 'sub_%'
ORDER BY name
LIMIT 20;
```

```sql
-- Shell-style star is accepted too
SELECT name, kind
FROM grep
WHERE pattern = 'Zw*'
LIMIT 20;
```

---

## Common Workflows

### Find candidate functions by name

```sql
SELECT name, addr
FROM grep
WHERE pattern = 'main%' AND kind = 'function'
ORDER BY name;
```

### Resolve imported APIs

```sql
SELECT module, name, addr
FROM imports
WHERE name LIKE 'CreateFile%'
ORDER BY module, name;
```

### Find types by convention

```sql
SELECT name, kind, ordinal, full_name
FROM grep
WHERE pattern = 'EH%' AND kind IN ('struct', 'enum')
ORDER BY kind, name;
```

### Find members under a parent type

```sql
SELECT name, parent_name, ordinal
FROM grep
WHERE pattern = 'flag%' AND kind = 'member'
ORDER BY parent_name, name
LIMIT 30;
```

### Join into richer function metadata

```sql
SELECT g.name, f.size, f.prototype
FROM grep g
JOIN funcs f ON f.addr = g.addr
WHERE g.pattern = 'sub_%' AND g.kind = 'function'
ORDER BY f.size DESC
LIMIT 20;
```

### Pivot from discovery into xrefs

```sql
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

## Compare With Other Search Surfaces

- Use `grep` for named entities discovered by IDA.
- Use `strings` when you need literal string contents.
- Use `byte_search` when you need raw bytes or opcode patterns.
- Use `xrefs` after discovery when the real question is "who references this?"

---

## Failure and Recovery

- Too many hits:
  add `kind = ...`, tighten the prefix, or switch from plain text to a more specific wildcard pattern.
- No hits for an expected symbol:
  broaden the pattern, try a contains search, or pivot to `imports` if the target may only exist as an imported API.
- Need to search for comments, pseudocode text, or string contents:
  `grep` is the wrong surface; pivot to `strings`, decompiler tables, or other domain tables.
- Need bytes/opcodes:
  use `byte_search` instead of `grep`.

