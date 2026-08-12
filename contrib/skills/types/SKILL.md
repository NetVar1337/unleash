---
name: types
description: "Use for IDA types: create/modify/apply structs, unions, enums, typedefs, C declarations."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: imported
  upstream: C:\Users\Admin\.agents\skills\types\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\types\SKILL.md

This skill is the **authoritative reference** for IDA's type system as exposed through idasql. For annotation workflows that use types, see `annotations`. For decompiler-specific type interactions (ctree, lvars, union selection, numform), see `decompiler`.

---

## Trigger Intents

Use this skill when user asks to:
- create/edit structs, unions, enums, typedefs
- inspect function prototype argument types
- resolve hidden pointer/typedef behavior
- apply or refine recovered data models

Route to:
- `decompiler` for expression-level type context
- `annotations` for applying and documenting type decisions
- `re-source` for recursive structure-recovery workflows

---

## Do This First (Warm-Start Sequence)

```sql
-- 1) Inventory local types
SELECT ordinal, name, kind, size
FROM types
ORDER BY ordinal
LIMIT 30;

-- 2) Large/high-signal structs
SELECT name, size
FROM types
WHERE is_struct = 1
ORDER BY size DESC
LIMIT 20;

-- 3) Prototype introspection sample
SELECT type_name, arg_index, arg_name, arg_type
FROM types_func_args
WHERE arg_index >= 0
LIMIT 40;
```

Interpretation guidance:
- Start with inventory and prioritize large/high-fanout types.
- Use `types_func_args` resolved fields for typedef-aware reasoning.

---

## Failure and Recovery

- Type insert/update failed:
  - Validate declaration syntax and target ordinal/name existence.
- Conflicting/incomplete type picture:
  - Correlate with `decompiler` (`ctree_lvars`, call args) before committing changes.
- Unexpected disassembly rendering:
  - Re-check operand format settings and applied declarations.
- Use the real surface names — common mistakes:
  - Members live in **`types_members`** (not `struct_members`); enum constants in
    **`types_enum_values`** (not `enum_members`).
  - There is no `print_type()` / `exec_python()` SQL function — render a type with
    `SELECT definition FROM types WHERE ...`; run Python via the `idapython` skill.
  - IDA 9 has no `ida_struct`; structs/unions are edited through `types` /
    `types_members` (see "Fixed layout, gaps & recovery").
  - To find references to a field, use **`struct_member_xrefs`**, not a
    `LIKE '%field%'` scan of `instructions.disasm` / `instruction_operands.text`.

---

## Handoff Patterns

1. `types` -> `decompiler` to validate semantic effect in pseudocode.
2. `types` -> `annotations` for naming/comments on newly typed fields.
3. `types` -> `re-source` for multi-function struct refinement.

---

## Type Tables

### local_types

Local type declarations as stored in the database.
Use this for quick inventory/filtering of local types; use `types*` tables for deeper editing workflows.

| Column | Type | Description |
|--------|------|-------------|
| `ordinal` | INT | Type ordinal (local type ID) |
| `name` | TEXT | Type name |
| `type` | TEXT | Declared type text |
| `is_struct` | INT | 1=struct |
| `is_enum` | INT | 1=enum |
| `is_typedef` | INT | 1=typedef |

```sql
-- Quick local type inventory
SELECT ordinal, name, type FROM local_types ORDER BY ordinal LIMIT 50;
```

For canonical schema and owner mapping, see `../connect/references/schema-catalog.md` (`local_types`).

### types

All local type definitions. Supports INSERT (create struct/union/enum), UPDATE, DELETE, and local type folder organization through `folder_path`.

| Column | Type | Description |
|--------|------|-------------|
| `ordinal` | INT | Type ordinal (unique identifier) |
| `name` | TEXT | Type name |
| `folder_path` | TEXT | RW local type folder relative to Local Types root; `NULL`/`''` means root |
| `full_path` | TEXT | RO full dirtree path, including the type name |
| `size` | INT | Size in bytes |
| `kind` | TEXT | struct/union/enum/typedef/func |
| `is_struct` | INT | 1=struct |
| `is_union` | INT | 1=union |
| `is_enum` | INT | 1=enum |

```sql
-- List all structs
SELECT ordinal, name, size FROM types WHERE is_struct = 1 ORDER BY size DESC;

-- List all enums
SELECT ordinal, name FROM types WHERE is_enum = 1;

-- Find types by name pattern
SELECT * FROM types WHERE name LIKE '%CONTEXT%';

-- Review type organization
SELECT ordinal, name, folder_path
FROM types
WHERE folder_path LIKE 'idasql/types/%'
ORDER BY folder_path, name;
```

**Performance.** `types WHERE ordinal = ?` and `types WHERE name = ?` are direct
lookups (resolve and render a single type). `types WHERE name LIKE 'prefix%'` is
optimized too — it scans names cheaply and only renders the matches. Prefer these
over `name LIKE '%substr%'`, which has no shortcut and renders every type
(slow on large IDBs); the same applies to an unfiltered `SELECT * FROM types`. If
a broad query is cut off by the query timeout, it returns cleanly and the next
query still works — narrow it with an exact name/ordinal or a `'prefix%'` filter.

### Organizing Local Types

Use `types.folder_path` for type recovery buckets and `dirtree_folders` for empty folder lifecycle. This is useful when recovered layouts move from draft to verified states.

```sql
INSERT INTO dirtree_folders(tree, path)
VALUES ('local_types', 'idasql/types/recovered');

UPDATE types
SET folder_path = 'idasql/types/recovered'
WHERE name IN ('MY_HEADER', 'COMMAND_RECORD');

UPDATE types
SET folder_path = 'idasql/types/verified'
WHERE name = 'MY_HEADER';

UPDATE types SET folder_path = NULL WHERE name = 'MY_HEADER';
```

For raw browsing across all IDA dirtrees, use `dirtree_entries`; for normal type organization, prefer `types.folder_path`. Folder writes use relative `/` paths. IDASQL rejects `.`/`..`, duplicate separators, backslashes, non-empty folder deletes, and folder renames whose destination already exists.

### local_type_bookmarks

Local-type (Local Types view) bookmarks, backed by the `bookmarks_t` store and
folder-aware via the `DIRTREE_LTYPES_BOOKMARKS` dirtree. Full CRUD:

| Column | Type | RW | Description |
|--------|------|----|-------------|
| `slot` | INT | R | Bookmark slot in the store |
| `ordinal` | INT | R | Local type ordinal |
| `type_name` | TEXT | R | Type name resolved from the ordinal |
| `description` | TEXT | RW | Bookmark description |
| `inode` | INT | R | Dirtree inode (`-1` if not linked into the dirtree) |
| `folder_path` | TEXT | RW | Folder path; `NULL`/`''` means root |
| `full_path` | TEXT | R | Full dirtree path |

```sql
-- Create a bookmark on a local type
INSERT INTO local_type_bookmarks(ordinal, description)
SELECT ordinal, 'review this struct' FROM types WHERE name = 'MY_STRUCT';

-- Edit / list / delete (derive the ordinal from the type name)
UPDATE local_type_bookmarks SET description = 'done'
WHERE ordinal = (SELECT ordinal FROM types WHERE name = 'MY_STRUCT');
SELECT slot, ordinal, type_name, description FROM local_type_bookmarks;
DELETE FROM local_type_bookmarks
WHERE ordinal = (SELECT ordinal FROM types WHERE name = 'MY_STRUCT');
```

Note: a freshly INSERTed bookmark is created in the `bookmarks_t` store but is
not auto-linked into the dirtree (so `folder_path` is `NULL` until linked), and
folder moves currently require an already-linked bookmark. Reliably mapping a
store slot back to its dirtree inode (to auto-link on INSERT) is a known
limitation.

### Creating Types

```sql
-- Create a struct
INSERT INTO types (name, kind) VALUES ('MY_HEADER', 'struct');

-- Create a union
INSERT INTO types (name, kind) VALUES ('PARAM_UNION', 'union');

-- Create an enum
INSERT INTO types (name, kind) VALUES ('CMD_TYPE', 'enum');

-- Verify creation (get the assigned ordinal)
SELECT ordinal, name, kind FROM types WHERE name = 'MY_HEADER';
```

### Deleting Types

```sql
-- Delete a type by name
DELETE FROM types WHERE name = 'MY_HEADER';

-- Delete by ordinal (derive it from the name to avoid a magic number)
DELETE FROM types WHERE ordinal = (SELECT ordinal FROM types WHERE name = 'MY_HEADER');
```

### Refining Existing Types Safely

When a type already exists, prefer targeted `types_members` edits over
re-importing a whole declaration. Member rename/retype/comment/delete operations
use IDA's UDM APIs against the existing local type ordinal, so dependent
prototypes and member type references keep pointing at the same type ID.

```sql
-- Capture the stable local type ID first
SELECT ordinal, name, kind FROM types WHERE name = 'MY_HEADER';

-- Rename and retype in place (ordinal derived from the type name)
UPDATE types_members
SET member_name = 'payload_len'
WHERE type_ordinal = (SELECT ordinal FROM types WHERE name = 'MY_HEADER') AND member_name = 'len';

UPDATE types_members
SET member_type = 'unsigned __int16'
WHERE type_ordinal = (SELECT ordinal FROM types WHERE name = 'MY_HEADER') AND member_name = 'payload_len';

-- Move a struct member by byte offset; this fixes the struct layout so IDA
-- preserves the explicit offset.
UPDATE types_members
SET offset = 16
WHERE type_ordinal = (SELECT ordinal FROM types WHERE name = 'MY_HEADER') AND member_name = 'payload_len';

-- Add a member into an existing fixed-layout gap
INSERT INTO types_members (type_ordinal, member_index, member_name, member_type, offset, comment)
SELECT ordinal, 2, 'flags', 'unsigned short', 8, 'inserted in place' FROM types WHERE name = 'MY_HEADER';
```

Notes:
- Key writes by `type_ordinal` plus `member_name` or `member_index`; read back
  the row after a write.
- `member_type` accepts member type text such as `int`, `void *`, `char[64]`,
  and named local types. Invalid type text fails instead of silently falling
  back to `int`.
- `offset` / `offset_bits` currently support byte-aligned struct moves. Union
  member offsets remain zero.
- Use `parse_decls()` to seed new related declarations or perform an intentional
  whole-definition redeclaration; for fine refinements of an existing recovered
  type, table edits are safer.
- For compound types, read `member_type_ordinal` before and after editing the
  referenced type. It should keep pointing at the same local type ordinal.

```sql
-- Verify dependent type references stay attached to the same ordinal
SELECT h.member_name, h.member_type_ordinal, t.ordinal AS target_ordinal
FROM types_members AS h
JOIN types AS t ON t.name = 'MY_HEADER'
WHERE h.type_name = 'MY_PACKET'
  AND h.member_name = 'header';
```

---

### types_members

Structure and union members. Supports INSERT, UPDATE, and DELETE.

| Column | Type | Description |
|--------|------|-------------|
| `type_ordinal` | INT | Parent type ordinal |
| `type_name` | TEXT | Parent type name |
| `member_index` | INT | Member index within the parent type |
| `member_name` | TEXT | Member name |
| `offset` | INT | Byte offset |
| `offset_bits` | INT | Bit offset |
| `size` | INT | Member size |
| `size_bits` | INT | Member size in bits |
| `member_type` | TEXT | Type string (e.g., `int`, `void *`, `char[256]`) |
| `comment` | TEXT | Member comment |
| `mt_is_ptr` | INT | 1=pointer |
| `mt_is_array` | INT | 1=array |
| `mt_is_struct` | INT | 1=embedded struct |
| `member_type_ordinal` | INT | Referenced local type ordinal, or -1 |

```sql
-- View members of a struct
SELECT member_name, member_type, offset, size
FROM types_members WHERE type_name = 'MY_HEADER'
ORDER BY offset;

-- Add members (member_type supports: int, void *, char[64], etc.).
-- Derive the ordinal from the type name so no magic ordinal is needed.
INSERT INTO types_members (type_ordinal, member_name, member_type)
SELECT ordinal, 'magic', 'unsigned int' FROM types WHERE name = 'MY_HEADER';
INSERT INTO types_members (type_ordinal, member_name, member_type)
SELECT ordinal, 'data_ptr', 'void *' FROM types WHERE name = 'MY_HEADER';

-- Insert at an explicit byte offset; offset_bits may be used for exact bits
INSERT INTO types_members (type_ordinal, member_name, member_type, offset)
SELECT ordinal, 'reserved', 'char[16]', 16 FROM types WHERE name = 'MY_HEADER';

-- Rename/retype/move a member in place
UPDATE types_members SET member_name = 'signature'
WHERE type_ordinal = (SELECT ordinal FROM types WHERE name = 'MY_HEADER') AND member_name = 'magic';
UPDATE types_members SET member_type = 'DWORD'
WHERE type_ordinal = (SELECT ordinal FROM types WHERE name = 'MY_HEADER') AND member_name = 'signature';
UPDATE types_members SET offset = 32
WHERE type_ordinal = (SELECT ordinal FROM types WHERE name = 'MY_HEADER') AND member_name = 'data_ptr';

-- Delete a member. DELETE leaves a gap (preserves the following members' offsets),
-- like IDA's Undefine — it never shifts them; see "Fixed layout, gaps & recovery".
DELETE FROM types_members
WHERE type_ordinal = (SELECT ordinal FROM types WHERE name = 'MY_HEADER') AND member_name = 'reserved';
```

---

### Fixed layout, gaps & recovery

A struct has two layout modes. **Auto-layout** (default): IDA recomputes member
offsets — repacking can shift fields and change the size. **Fixed-layout**
(`types.is_fixed = 1`): every offset is frozen, the size can be pinned, and empty
space is exposed as gaps. Use fixed layout to refine a recovered struct in place
without disturbing the bytes you've already mapped — this replaces the old
`parse_decls()` full-rebuild workaround for ordinary edits.

| Surface | Meaning |
|---|---|
| `types.is_fixed` (writable) | `1` freezes member offsets; `0` returns to auto-layout — IDA repacks/compacts only on the **fixed→auto transition** (may shrink), so `0` on an already-auto struct is a no-op |
| `types.size` (writable) | pin a **fixed** struct's total size (errors on an auto struct) |
| `types_members.is_gap` | `1` if the member is an IDA gap placeholder (free space) |
| `type_gaps` | one row per free byte range in a struct: `gap_offset`, `gap_size` (requires a `type_ordinal`/`type_name` filter) |

`DELETE` leaves the slot as a **gap** (the following members' offsets and the struct
size are preserved — *non-collapsing*, like IDA's Undefine) on **any** struct, auto
or fixed — it never shifts the members after it. `INSERT` at an offset inside a gap
absorbs it without moving neighbors. An `INSERT` whose explicit `offset` overlaps an
existing (non-gap) member is **rejected** ("offset range overlaps an existing
member") — it never overwrites a real field; target a free `type_gaps` range, or omit
`offset` to append. To deliberately **compact** and shrink a struct (drop the gaps and
repack), set `is_fixed = 1` then `is_fixed = 0`: IDA only repacks on the fixed→auto
transition, so a plain `is_fixed = 0` on an already-auto struct is a no-op.

```sql
-- Freeze layout, then see where the free space is
UPDATE types SET is_fixed = 1 WHERE name = 'VARBIND_VARIABLE';
SELECT gap_offset, gap_size FROM type_gaps WHERE type_name = 'VARBIND_VARIABLE';
SELECT member_name, "offset" FROM types_members
WHERE type_name = 'VARBIND_VARIABLE' AND is_gap = 1;

-- Absorb a gap: drop a real field into free space (neighbors stay put).
-- Derive the ordinal from the name so the snippet needs no magic number.
INSERT INTO types_members (type_ordinal, member_name, member_type, "offset")
SELECT ordinal, 'handler_cookie', 'int', 28 FROM types WHERE name = 'VARBIND_VARIABLE';

-- Non-collapsing delete: the slot becomes a gap; later offsets don't shift
DELETE FROM types_members
WHERE type_ordinal = (SELECT ordinal FROM types WHERE name = 'VARBIND_VARIABLE')
  AND member_name = 'scratch';

-- Pin the total size of a fixed struct
UPDATE types SET size = 68 WHERE name = 'VARBIND_VARIABLE';

-- Deliberately compact (remove gaps, shrink): toggle fixed off
UPDATE types SET is_fixed = 0 WHERE name = 'VARBIND_VARIABLE';
```

**Recovery (prefer this over `parse_decls()` for refinements).** If a fixed struct
got into a bad state from earlier edits: set `is_fixed = 1`, pin `size`, inspect
`type_gaps`/`is_gap` to find free space, then add/move/delete members
incrementally. Reserve `parse_decls()` full replacement for building a type from
scratch or a wholesale redefinition — not for ordinary member refinements.

---

### types_enum_values

Enum constant values. Supports INSERT, UPDATE, and DELETE.

| Column | Type | Description |
|--------|------|-------------|
| `type_ordinal` | INT | Enum type ordinal |
| `type_name` | TEXT | Enum name |
| `value_name` | TEXT | Constant name |
| `value` | INT | Constant value |

```sql
-- View enum values
SELECT value_name, value FROM types_enum_values
WHERE type_name = 'CMD_TYPE'
ORDER BY value;

-- Add enum values (ordinal derived from the enum name; optional comment column)
INSERT INTO types_enum_values (type_ordinal, value_name, value)
SELECT ordinal, 'CMD_INIT', 0 FROM types WHERE name = 'CMD_TYPE';
INSERT INTO types_enum_values (type_ordinal, value_name, value)
SELECT ordinal, 'CMD_READ', 1 FROM types WHERE name = 'CMD_TYPE';

-- Rename / change value / comment / delete enum values in place
UPDATE types_enum_values SET value_name = 'CMD_OPEN'
WHERE type_ordinal = (SELECT ordinal FROM types WHERE name = 'CMD_TYPE') AND value_name = 'CMD_INIT';
UPDATE types_enum_values SET value = 11, comment = 'opens a stream'
WHERE type_ordinal = (SELECT ordinal FROM types WHERE name = 'CMD_TYPE') AND value_name = 'CMD_OPEN';
DELETE FROM types_enum_values
WHERE type_ordinal = (SELECT ordinal FROM types WHERE name = 'CMD_TYPE') AND value_name = 'CMD_READ';
```

Enum value edits update the existing enum type ordinal. Prefer this path over
redeclaring the enum when the enum is already used by structs, prototypes,
operand representations, or decompiler numforms.

#### Bitfield (bitmask) enums

A **bitmask enum** has flag members that combine with OR; a matching operand value
renders as `A or B` (disassembly) / `A | B` (pseudocode) instead of a number. Mark
the enum with the writable `types.is_bitmask` column and set its byte width with the
writable `types.size` column. Members are added the normal way (flat flags — each
value is its own mask).

```sql
-- 1) create the enum, make it a 4-byte bitmask
INSERT INTO types(name, kind) VALUES ('FILE_FLAGS', 'enum');
UPDATE types SET is_bitmask = 1, size = 4 WHERE name = 'FILE_FLAGS';

-- 2) add single-bit flag members
INSERT INTO types_enum_values(type_ordinal, value_name, value)
  SELECT ordinal, 'FF_READ', 1 FROM types WHERE name='FILE_FLAGS';   -- + FF_WRITE 2, FF_EXEC 4

-- 3) apply it to an operand whose immediate is a combination (e.g. 3 = READ|WRITE)
UPDATE instructions SET operand1_format_spec = 'enum:FILE_FLAGS' WHERE addr = 0x401050;

-- now: SELECT disasm FROM instructions WHERE addr=0x401050;  -> mov eax, FF_READ or FF_WRITE
--      SELECT line   FROM pseudocode   WHERE func_addr=0x401000; -> ... == (FF_READ | FF_WRITE)
```

`is_bitmask` applies to enums only; `size` must be 1/2/4/8. The flag + width persist
across later value edits. Decompiler propagation is operand-dependent (a `cmp`/`test`
or call-argument immediate is the reliable case). Omitting `value` when inserting a
bitmask member auto-assigns the **next free single-bit mask** (`1`, `2`, `4`, …) —
`0` is not a valid mask, so a value-less insert never yields it. The auto-mask is
bounded by the enum width: a 1-byte bitmask only has `0x01..0x80`, so once every
in-width bit is used a value-less insert **fails** ("no free single-bit mask remains
within the enum width") — widen `types.size` or pass an explicit `value`.

---

### types_func_args

Function prototype arguments with deep type classification.

| Column | Type | Description |
|--------|------|-------------|
| `type_ordinal` | INT | Function type ordinal |
| `type_name` | TEXT | Function type name |
| `arg_index` | INT | Argument index (-1 = return type, 0+ = args) |
| `arg_name` | TEXT | Argument name |
| `arg_type` | TEXT | Argument type string |
| `calling_conv` | TEXT | Calling convention (on return row only) |

#### Surface-Level Type Classification

Literal type as written — what you see in the declaration:

| Column | Description |
|--------|-------------|
| `is_ptr` | 1 if pointer type |
| `is_int` | 1 if exactly `int` |
| `is_integral` | 1 if int-like (int, long, short, char, bool) |
| `is_float` | 1 if float/double |
| `is_void` | 1 if void |
| `is_struct` | 1 if struct/union |
| `is_array` | 1 if array |
| `ptr_depth` | Pointer depth (int** = 2) |
| `base_type` | Type with pointers stripped |

#### Resolved Type Classification

After typedef resolution — what the type actually is:

| Column | Description |
|--------|-------------|
| `is_ptr_resolved` | 1 if resolved type is pointer |
| `is_int_resolved` | 1 if resolved type is exactly int |
| `is_integral_resolved` | 1 if resolved type is int-like |
| `is_float_resolved` | 1 if resolved type is float/double |
| `is_void_resolved` | 1 if resolved type is void |
| `ptr_depth_resolved` | Pointer depth after resolution |
| `base_type_resolved` | Resolved type with pointers stripped |

This dual classification is critical for typedef-aware queries. For example, `HANDLE` appears as non-pointer at surface level but resolves to `void *`.

```sql
-- Typedefs that hide pointers (HANDLE, HMODULE, etc.)
SELECT DISTINCT type_name, arg_type, base_type_resolved
FROM types_func_args
WHERE is_ptr = 0 AND is_ptr_resolved = 1;

-- Functions with struct parameters
SELECT type_name, arg_name, arg_type FROM types_func_args
WHERE arg_index >= 0 AND is_struct = 1;
```

For more `types_func_args` query patterns (string parameters, pointer counts, return type filters), see [references/type-patterns.md](references/type-patterns.md).

---

## Type Views

Convenience views for filtering types:

| View | Description |
|------|-------------|
| `types_v_structs` | `SELECT * FROM types WHERE is_struct = 1` |
| `types_v_unions` | `SELECT * FROM types WHERE is_union = 1` |
| `types_v_enums` | `SELECT * FROM types WHERE is_enum = 1` |
| `types_v_typedefs` | `SELECT * FROM types WHERE is_typedef = 1` |
| `types_v_funcs` | `SELECT * FROM types WHERE is_func = 1` |
| `types_v_inheritance` | Struct/class inheritance relationships (baseclasses from `types_members`) |

### `types_v_inheritance` view

Shows struct/class inheritance relationships extracted from baseclass members.

| Column | Type | Description |
|--------|------|-------------|
| `derived_ordinal` | INT | Ordinal of the derived type |
| `derived_name` | TEXT | Name of the derived type |
| `base_type_name` | TEXT | Name of the base type |
| `base_ordinal` | INT | Ordinal of the base type |
| `base_offset` | INT | Byte offset of the base within the derived type |

```sql
-- Find base classes of a type
SELECT * FROM types_v_inheritance WHERE derived_name = 'MyClass';

-- Recursive ancestors
WITH RECURSIVE ancestors(name, depth) AS (
    SELECT base_type_name, 1 FROM types_v_inheritance WHERE derived_name = 'MyClass'
    UNION ALL
    SELECT i.base_type_name, a.depth + 1
    FROM types_v_inheritance i JOIN ancestors a ON i.derived_name = a.name
    WHERE a.depth < 10
)
SELECT * FROM ancestors;
```

---

## Importing C Declarations (parse_decls)

`parse_decls(text)` imports C declarations into the local type library. It is
best for seeding related types, typedefs, enums, and whole declarations. For
small refinements to an existing recovered type, prefer `types_members` updates
so the local type ordinal stays stable and existing references do not need to be
rebuilt.

```sql
-- Import a simple struct
SELECT parse_decls('
struct MY_HEADER {
    unsigned int magic;
    unsigned int version;
    unsigned int size;
    void *data;
};
');

-- Import with pragmas for packing (enums, typedefs, nested unions)
SELECT parse_decls('
#pragma pack(push, 1)
typedef enum operations_e { op_empty=0, op_open=11, op_read=22 } operations_e;
typedef union command_payload_u { void *ptr; unsigned __int64 qword; } command_payload_u;
typedef struct command_t {
    operations_e cmd_id;
    command_payload_u payload;
    unsigned __int64 ret;
} command_t;
#pragma pack(pop)
');

-- Verify imported types
SELECT name, kind, size FROM types WHERE name IN ('command_t', 'operations_e');
```

For a full multi-struct `parse_decls` example with nested unions, see [references/type-patterns.md](references/type-patterns.md).

---

## Applying Types to Functions and Variables

### Function Prototypes

```sql
-- Apply type to function via prototype column
UPDATE funcs SET prototype = 'void __fastcall exec_command(command_t *cmd);'
WHERE addr = 0x140001BD0;

-- Apply/replace the type at any mapped address
INSERT INTO applied_types(addr, decl)
VALUES (0x140001BD0, 'void __fastcall exec_command(command_t *cmd);');

-- Read current type at address
SELECT decl, ordinal, type_name
FROM applied_types
WHERE addr = 0x140001BD0;

-- Address equality also accepts numeric strings and symbol names
UPDATE applied_types
SET decl = 'void __fastcall exec_command(command_t *cmd);'
WHERE addr = 'exec_command';

-- Clear type (reset to auto-detected)
DELETE FROM applied_types WHERE addr = 0x140001BD0;

-- Re-decompile to see effect
SELECT decompile(0x140001BD0, 1);
```

`applied_types` point lookup is intentionally write-friendly: `WHERE addr = X`
returns one mapped row even when no declaration is applied yet, with `decl`,
`ordinal`, and `type_name` as NULL. Range scans return only addresses that
currently have applied type information.

### Local Variables

```sql
-- Change local variable type
UPDATE ctree_lvars SET type = 'MY_STRUCT *'
WHERE func_addr = 0x401000 AND idx = 0;

-- Change and verify
SELECT decompile(0x401000, 1);
SELECT idx, name, type FROM ctree_lvars
WHERE func_addr = 0x401000 AND idx = 0;
```

### Call Sites

Use call-site typing for indirect calls when function prototypes and local-variable types still leave a specific call under-typed.

```sql
-- Discover indirect call sites first
SELECT call_addr, target_op, target_var_name, arg_count
FROM ctree_v_indirect_calls
WHERE func_addr = 0x140001BD0
ORDER BY call_addr;

-- Apply a prototype to one call site
UPDATE disasm_calls
SET callee_type = 'int __fastcall emit_message(const char *name, const char *target, int flag, const char *tag);'
WHERE addr = 0x140001C3E;

-- Verify the persisted call-site typing
SELECT callee_type FROM disasm_calls WHERE addr = 0x140001C3E;
SELECT call_arg_addrs(0x140001C3E);
SELECT decompile(0x140001BD0, 1);
```

### Typing Surfaces Matrix

| Surface | Scope | Semantic vs render-only | Typical use |
|---------|-------|-------------------------|-------------|
| `UPDATE funcs SET prototype = ...` / `applied_types` | Function/global address | Semantic | Give a function or global the right declared type |
| `UPDATE ctree_lvars SET type = ...` | One decompiled local/arg | Semantic | Clean up local pointer/struct inference |
| `UPDATE disasm_calls SET callee_type = ... WHERE addr = ...` | One call site | Semantic | Fix an indirect call when the callee prototype must be explicit |
| `instructions.operand*_format_spec` | One disassembly operand | Render-only | Show enums/struct offsets in listing output |
| `set_union_selection*` | One decompiler expression | Render-only | Choose a union arm for nicer pseudocode |
| `set_numform*` | One decompiler expression operand | Render-only | Change numeric rendering without changing base type |

### Names

```sql
-- Set a name at address (or replace any existing name at that EA)
INSERT INTO names(addr, name) VALUES (0x402000, 'g_config');
-- Equivalent: rename in place
UPDATE names SET name = 'g_config' WHERE addr = 0x402000;
```

Note: `INSERT` and `UPDATE` against `names` both call IDA's `set_name(addr, name, SN_CHECK)`. IDA permits only one name per address, so `INSERT` at an already-named EA **replaces**. `SN_CHECK` also auto-disambiguates if the new name string conflicts globally (`foo` → `foo_0`); read back the row to see what was actually stored.

---

## Struct Offset Representation in Disassembly

The `instructions` table `operand*_format_spec` column applies struct offset display to disassembly operands:

```sql
-- Apply struct-offset: makes `[rax+10h]` display as `[rax+MY_STRUCT.field_name]`
UPDATE instructions SET operand0_format_spec = 'stroff:MY_STRUCT,delta=0'
WHERE addr = 0x401030;

-- Nested member path: separate type names with '/'
UPDATE instructions SET operand0_format_spec = 'stroff:OUTER_T/INNER_T'
WHERE addr = 0x401030;

-- sizeof: an immediate equal to sizeof(STRUCT) renders as `size STRUCT`
UPDATE instructions SET operand1_format_spec = 'sizeof:MY_STRUCT'
WHERE addr = 0x4015EB;

-- Apply enum: `enum:<ENUM_NAME>`; clear back to plain: `clear`
UPDATE instructions SET operand1_format_spec = 'enum:MY_ENUM'
WHERE addr = 0x401020;
```

Number/offset/forced/char display forms (`hex`/`dec`/`oct`/`bin`, `char`,
`offset[:base]`, `forced:<text>`, plus `,signed`/`,bnot` modifiers) are also
applied through `operand*_format_spec` — see the `disassembly` skill for the
full vocabulary.

---

## Enum/Union Rendering in Decompiled Code

For numform helpers (`set_numform*`) and union selection helpers (`set_union_selection*`), see `decompiler` skill.

`disasm_calls.callee_type` belongs on the semantic side of the fence: it affects call analysis, unlike render-only enum/union formatting helpers.

---

## Performance Rules

| Table | Architecture | Key Constraint | Notes |
|-------|-------------|----------------|-------|
| `types` | Cached | `ordinal` (optional) | Full cache rebuilt on demand; usually fast (<1000 types) |
| `types_members` | Cached | `type_ordinal` | O(1) lookup with constraint; without it iterates all types |
| `types_enum_values` | Cached | `type_ordinal` | O(1) lookup with constraint |
| `types_func_args` | Cached | `type_ordinal` | O(1) lookup with constraint |

**Key rules:**
- `type_ordinal` constraint pushdown gives O(1) access to a single type's members, enum values, or func args.
- Without constraint, these tables iterate all local types. This is usually fast (most binaries have <1000 local types), but prefer filtered queries when you know the target.
- Type views (`types_v_structs`, etc.) are pre-filtered — use them for categorical queries.
- `parse_decls()` is the fastest way to seed multiple types at once (single call vs multiple INSERTs).

---

## See Also

- `xrefs` — `struct_member_xrefs` finds references TO a struct/union member
  (filter by `type_name`/`type_ordinal`/`member_id`); the right tool after you
  resolve a type/member here, instead of disasm-text scans.
- `annotations` — apply type via `funcs.prototype` / `applied_types`; combine with renames and comments.
- `decompiler` — ctree consumes applied types; union selection, numform, mutation loop.
- `re-source` — structure recovery methodology from offset casts.

---

## Additional Resources

- For complete type workflow examples and advanced CTE patterns: [references/type-patterns.md](references/type-patterns.md)

