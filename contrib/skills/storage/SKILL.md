---
name: storage
description: "Use to persist key-value metadata in IDA databases via netnode_kv."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: imported
  upstream: C:\Users\Admin\.agents\skills\storage\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\storage\SKILL.md

## netnode_kv

Persistent key-value store backed by IDA netnodes. Data is saved inside the IDB automatically. Supports full CRUD and O(1) key lookup via `WHERE key = '...'`.

| Column | Type | Writable | Description |
|--------|------|----------|-------------|
| `key` | TEXT | — | Unique key (identity, read-only) |
| `value` | TEXT | Yes | Arbitrary-length value (blob storage) |

```sql
-- Store a value
INSERT OR REPLACE INTO netnode_kv(key, value) VALUES('author', 'alice');

-- Read by key (O(1) lookup)
SELECT value FROM netnode_kv WHERE key = 'author';

-- List all entries
SELECT * FROM netnode_kv;

-- Update a value
UPDATE netnode_kv SET value = '2.0' WHERE key = 'version';

-- Delete an entry
DELETE FROM netnode_kv WHERE key = 'author';
```

### Use Cases

- **Session state**: Track analysis progress across sessions (e.g., which functions have been annotated)
- **Metadata**: Store custom metadata like analyst name, analysis date, notes
- **Bookkeeping**: Track which functions have been re-sourced, annotated, or reviewed
- **Configuration**: Store per-database analysis settings

```sql
-- Track analysis progress
INSERT OR REPLACE INTO netnode_kv(key, value) VALUES('annotated_funcs', '["main","init_config"]');

-- Update progress
UPDATE netnode_kv SET value = '["main","init_config","process_input"]'
WHERE key = 'annotated_funcs';

-- Read progress in a new session
SELECT value FROM netnode_kv WHERE key = 'annotated_funcs';
```

---

## Performance Rules

| Operation | Complexity | Notes |
|-----------|-----------|-------|
| `WHERE key = '...'` | O(1) | IDA's netnode `hashval_long()` — always use exact key lookup |
| `WHERE key LIKE 'prefix%'` | O(n) | Scans all entries; acceptable for small datasets |
| `SELECT * FROM netnode_kv` | O(n) | Full netnode scan; fine for typical use (dozens to hundreds of entries) |

**Key rules:**
- Exact key lookup (`WHERE key = '...'`) is O(1) — this is the preferred access pattern.
- Prefix scans (`LIKE 'prefix%'`) iterate all entries but are fast for typical netnode sizes.
- netnode_kv is stored inside the IDB file — it persists automatically with `save_database()`.

---

## Advanced Storage Patterns

### JSON-based progress tracking

Store structured analysis state as JSON for richer querying:

```sql
-- Store progress with structured metadata
INSERT OR REPLACE INTO netnode_kv(key, value)
VALUES('progress:overview', json_object(
    'total_funcs', (SELECT COUNT(*) FROM funcs),
    'named_funcs', (SELECT COUNT(*) FROM funcs WHERE name NOT LIKE 'sub_%'),
    'timestamp', datetime('now')
));

-- Read and parse progress
SELECT json_extract(value, '$.total_funcs') AS total,
       json_extract(value, '$.named_funcs') AS named,
       json_extract(value, '$.timestamp') AS ts
FROM netnode_kv WHERE key = 'progress:overview';
```

### Per-function annotation status tracking

Track which functions have been annotated and what was done:

```sql
-- Mark a function as annotated
INSERT OR REPLACE INTO netnode_kv(key, value)
VALUES('re_source:' || printf('0x%X', 0x401000),
       json_object('status', 'done', 'summary', 'DriverEntry init',
                    'analyst', 'alice', 'date', date('now')));

-- Find unannotated functions by joining with funcs
SELECT f.name, printf('0x%X', f.addr) AS addr
FROM funcs f
WHERE f.name NOT LIKE 'sub_%'
  AND NOT EXISTS (
    SELECT 1 FROM netnode_kv
    WHERE key = 're_source:' || printf('0x%X', f.addr)
  )
ORDER BY f.size DESC
LIMIT 20;
```

### Naming conventions for keys

Use a `namespace:entity:id` format for organized storage:

```
re_source:0x401000          → per-function annotation status
config:string_minlen        → analysis configuration
snapshot:2024-01-15          → point-in-time analysis snapshot
tag:crypto:0x401000         → function tags/categories
```

```sql
-- List all keys in a namespace
SELECT key, value FROM netnode_kv WHERE key LIKE 'tag:crypto:%';

-- Count entries per namespace
SELECT SUBSTR(key, 1, INSTR(key, ':') - 1) AS namespace,
       COUNT(*) AS entries
FROM netnode_kv
WHERE key LIKE '%:%'
GROUP BY namespace
ORDER BY entries DESC;
```

---

## See Also

- `annotations` — for analyst breadcrumbs that should live in the address space (comments, bookmarks). Use `netnode_kv` for cross-address session state.

