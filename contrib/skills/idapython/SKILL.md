---
name: idapython
description: "Use when SQL is insufficient and direct IDA SDK access via Python is needed."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: re
  upstream: C:\Users\Admin\.agents\skills\idapython\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\idapython\SKILL.md

## Python Execution SQL Functions

| Function | Description |
|----------|-------------|
| `idapython_snippet(code[, sandbox])` | Execute Python snippet and return captured output text |
| `idapython_file(path[, sandbox])` | Execute Python file and return captured output text |

### Runtime Guard

Python execution is disabled by default. Enable it with:

```sql
PRAGMA idasql.enable_idapython = 1;
```

Captured `print` output is **unbounded by default**. To bound a runaway or very
chatty snippet (so it can't exhaust memory / the response), set an optional byte cap;
`0` restores unbounded:

```sql
PRAGMA idasql.idapython_output_max = 65536;  -- cap captured output at 64 KiB
PRAGMA idasql.idapython_output_max = 0;      -- unbounded (default)
```

Output past the cap is dropped with a `...[idapython output truncated at N bytes]...` marker.

To confirm the current values of `enable_idapython` and `idapython_output_max` (and
the other runtime controls), `SELECT * FROM runtime_settings WHERE scope = 'idasql'` —
a read-only discovery view over the `PRAGMA idasql.*` surface. See the `connect` skill.

### Examples

```sql
SELECT idapython_snippet('print("hello from idapython")');
SELECT idapython_file('C:/temp/script.py');
SELECT idapython_snippet('counter = globals().get("counter", 0) + 1; print(counter)', 'alpha');
```

### Notes

- Disabled by default until pragma is enabled
- Python exceptions propagate as SQL errors
- `sandbox` isolates/persists Python globals by sandbox key

### Two Python Contexts (Important)

- **Host-side Python client** (outside IDA): use `requests.post(.../query, data=sql)` to send SQL over HTTP. The body may be one statement or a semicolon-separated script; every response uses the canonical `results[]` envelope (a single statement is an array of one — read `results[i].rows`). Use this for loops, bulk updates, and automation orchestration. See `connect` skill HTTP client patterns.
- **IDAPython via SQL** (inside IDA): use `idapython_snippet()` / `idapython_file()` when you need direct IDA SDK APIs in-process.

Example contrast:

```python
# Host-side Python (outside IDA): sends SQL over HTTP
import requests
requests.post("http://127.0.0.1:8080/query", data="SELECT COUNT(*) FROM funcs")
```

```sql
-- IDAPython (inside IDA): executes Python in IDA runtime
SELECT idapython_snippet('import idaapi; print(idaapi.get_kernel_version())');
```

### Sandbox Behavior

Each sandbox key creates an isolated Python namespace:
- Variables set in one sandbox are not visible in another
- The same sandbox key reuses its namespace across calls (state persists within a session)
- Without a sandbox key, code runs in the default global namespace

### Error Propagation

When a Python script raises an exception, it propagates as a SQL error:
```sql
-- This will return an error: "NameError: name 'undefined_var' is not defined"
SELECT idapython_snippet('print(undefined_var)');
```

---

## When to Use IDAPython vs SQL

| Use Case | Best Tool | Why |
|----------|-----------|-----|
| Query/filter/aggregate data | SQL | JOINs, CTEs, GROUP BY, window functions — SQL is purpose-built for this |
| Cross-table analysis | SQL | JOINing `funcs`, `xrefs`, `strings`, `ctree` is natural in SQL |
| Reporting and counting | SQL | COUNT, SUM, AVG, GROUP_CONCAT — no Python loop needed |
| Complex algorithms | IDAPython | Graph algorithms, custom pattern matching, ML pipelines |
| IDA SDK APIs not in idasql | IDAPython | Some IDA SDK features aren't exposed as SQL tables/functions |
| UI automation | IDAPython | Opening views, navigating cursor, triggering IDA actions |
| Existing scripts | IDAPython | Reuse existing `.py` scripts without rewriting in SQL |

**General rule:** Start with SQL. If you find yourself wanting nested loops, recursive algorithms, or IDA APIs that aren't exposed via idasql, reach for `idapython_snippet()` as a bridge.

---

## Practical Use Cases

### Run a custom analysis script

```sql
-- Enable Python execution first
PRAGMA idasql.enable_idapython = 1;

-- Run a script that collects custom metrics
SELECT idapython_snippet('
import idautils, idc
count = 0
for func_addr in idautils.Functions():
    if idc.get_func_attr(func_addr, idc.FUNCATTR_FLAGS) & 0x4:  # FUNC_LIB
        count += 1
print(f"Library functions: {count}")
');
```

### Access IDA SDK APIs not exposed through idasql

```sql
-- Example: get processor-specific register names
SELECT idapython_snippet('
import ida_idp
for i in range(ida_idp.ph_get_regnames().__len__()):
    name = ida_idp.ph_get_regnames()[i]
    if name:
        print(f"{i}: {name}")
');
```

### Bridge pattern: Python produces JSON, SQL processes it

When you need Python's power for extraction but SQL's power for analysis:

```sql
-- Python extracts data as JSON
SELECT idapython_snippet('
import json, idautils, idc
result = []
for addr in idautils.Functions():
    flags = idc.get_func_attr(addr, idc.FUNCATTR_FLAGS)
    if flags & 0x4:  # FUNC_LIB
        result.append({"addr": addr, "name": idc.get_func_name(addr)})
print(json.dumps(result))
');

-- Then process the JSON output in SQL using json_each()
-- (copy the output from above into the query)
```

---

## See Also

- Check the relevant SQL skill first (`disassembly`, `decompiler`, `types`, `data`, `xrefs`, `annotations`, `debugger`) — fall through to IDAPython only when no SQL surface exists for the task.
- `functions` — the SQL function catalog; verify a scalar/helper doesn't already exist before scripting.

