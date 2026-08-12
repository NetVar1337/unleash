---
name: annotations
description: "Use to edit IDA DBs: comments, renames, types, bookmarks, decompiler cleanup."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: re
  upstream: C:\Users\Admin\.agents\skills\annotations\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\annotations\SKILL.md

This skill is your guide for **editing** IDA databases through idasql. Use it whenever you need to annotate, rename, retype, comment, or otherwise modify decompiled output, disassembly, or type information.

---

## Trigger Intents

Use this skill when user asks to:
- add/edit comments (pseudocode, disassembly, function summaries)
- rename symbols or local variables
- apply types/enums/struct representations
- create/update bookmarks for investigation workflow
- make pseudocode read more like source
- prepare decompiled code for side-by-side review against source
- annotate a function end-to-end, not just add one comment

Route to:
- `decompiler` for analysis before editing
- `types` when declarations or struct models need construction
- `re-source` for recursive narrative/source recovery passes

---

## Do This First (Warm-Start Sequence)

```sql
-- 1) Confirm target row/function before editing
SELECT * FROM funcs WHERE addr = 0x401000;

-- 2) Inspect current comment state
SELECT addr, line, comment
FROM pseudocode
WHERE func_addr = 0x401000
LIMIT 30;

-- 3) Inspect existing disassembly comments
SELECT * FROM comments WHERE addr BETWEEN 0x401000 AND 0x401100;
```

Interpretation guidance:
- Never edit blind: validate exact row identity before mutation.
- Prefer deterministic keys (`func_addr + addr`, `idx`, `slot`) over fuzzy name matches.

---

## Failure and Recovery

- Update appeared to "do nothing":
  - Re-read target row, refresh decompile view, verify placement fields.
- Wrong target mutated:
  - Tighten predicate and re-run read-first step before retry.
- Enum/struct rendering did not change:
  - Confirm type/numform/union-selection support and target context.

---

## Handoff Patterns

1. `annotations` -> `decompiler` when semantic meaning is unclear.
2. `annotations` -> `types` when edits imply missing declarations.
3. `annotations` -> `re-source` when function-level notes must become recursive campaign notes.

---

## Annotate a Function Contract

Treat `annotate this function` as a full-function workflow, not a single comment operation.

Default behavior:
- inspect the current decompilation and local/label state
- apply types/prototypes that make the function read more like source
- rename locals, globals, and labels when they materially improve readability
- add targeted line comments where they help interpretation
- place functions into review/triage folders when the workflow creates durable buckets
- always finish with exactly one repeatable function comment summary on `funcs.rpt_comment`
- refresh with `decompile(addr, 1)` and verify the result

Use narrower intents only when the user asks narrowly:
- `add a comment` -> comment only
- `func-summary` / `function summary` -> summary only
- `annotate this function` -> full cleanup plus summary

Why the summary is mandatory:
- it orients a human reviewer immediately
- it creates semantic/searchable program knowledge for later whole-program understanding

---

## Folder-Aware Annotation Workflow

Use object-table `folder_path` columns to organize work as you annotate. Prefer `funcs.folder_path` for function-level work, `names.folder_path` for globals/labels, and `bookmarks.folder_path` for review breadcrumbs. `dirtree_entries` is the raw browser and `dirtree_folders` manages empty folders across all standard IDA dirtrees.

Common folders:
- `idasql/review/needs-types`
- `idasql/review/annotated`
- `idasql/review/verified`
- `idasql/triage/network`
- `idasql/triage/crypto`

```sql
-- Create a review bucket and move selected functions into it
INSERT INTO dirtree_folders(tree, path)
VALUES ('funcs', 'idasql/review/needs-types');

UPDATE funcs
SET folder_path = 'idasql/review/needs-types'
WHERE addr = 0x401000;

-- Mark a function as annotated while adding the required repeatable summary
UPDATE funcs
SET folder_path = 'idasql/review/annotated',
    rpt_comment = 'Parses command records, validates the opcode, and dispatches to command handlers.'
WHERE addr = 0x401000;

UPDATE names
SET folder_path = 'idasql/names/globals'
WHERE name LIKE 'g_%';

UPDATE bookmarks
SET folder_path = 'idasql/bookmarks/review'
WHERE description LIKE '%review%';

-- Review progress
SELECT folder_path, COUNT(*) AS functions
FROM funcs
WHERE folder_path LIKE 'idasql/%'
GROUP BY folder_path
ORDER BY folder_path;
```

Use `UPDATE ... SET folder_path = NULL WHERE ...` to move an object back to root. `DELETE FROM dirtree_folders` removes only empty folders. Folder writes use relative `/` paths and reject `.`/`..`, duplicate separators, backslashes, and renames to an already-existing destination.

---

## Editing Function Comments

True IDA function comments live on `funcs`, not on `pseudocode` and not on the address-level `comments` table.

Use:
- `funcs.comment` for the regular function comment
- `funcs.rpt_comment` for the repeatable function comment

Default function-summary behavior:
- treat singular `add function comment` as `UPDATE funcs SET rpt_comment = ...`
- use `funcs.comment` only when the user explicitly asks for a non-repeatable function comment
- use a heading-style `pseudocode` block comment only when the user explicitly asks for a decompiler note/summary rather than a true function comment

Canonical SQL pattern:

```sql
SELECT addr, name, comment, rpt_comment
FROM funcs
WHERE addr = 0x401000;

UPDATE funcs
SET rpt_comment = 'One-paragraph summary of what the function does, inputs/outputs, and key behavior.'
WHERE addr = 0x401000;
```

---

## Editing Decompiler Comments (Pseudocode)

The `pseudocode` table is the editing surface for decompiler comments. Use `decompile(addr)` to view; use the table to edit. For full table schema, see `decompiler` skill.

Important separation:
- `pseudocode.comment` edits Hex-Rays decompiler comments only.
- `pseudocode` writes never call `set_func_cmt()`.
- Use `funcs.comment` / `funcs.rpt_comment` for true function comments.

Writable columns: `comment`, `comment_placement`. Placements: `semi` (after `;`), `block1` (own line above), `block2`, `curly1`, `curly2`, `brace1`, `brace2`, `colon`, `case`, `else`, `do`, `asm`, `try`. An unrecognized placement string is silently coerced to `semi`.

Anchor guidance:
- Prefer a concrete pseudocode statement row, not a guessed function-entry row.
- If an `addr` maps to multiple pseudocode rows (`{`, statement, `}`), resolve a unique non-brace anchor first.
- Use `line_num` only to inspect candidate rows. Comment writes persist by `addr + comment_placement`; shared-`addr` rows need extra care, so do not assume every displayed shared-`addr` row is independently writable.

Inspect anchors before writing:

```sql
SELECT line_num, addr, line, comment
FROM pseudocode
WHERE func_addr = 0x401000
ORDER BY line_num;
```

```sql
-- The example UPDATEs below assume 0x401020 is an already resolved writable
-- non-brace anchor from the inspection query above; do not substitute func_addr.
-- Edit: Add inline comment to decompiled code
UPDATE pseudocode SET comment_placement = 'semi',
                      comment = 'buffer overflow here'
WHERE func_addr = 0x401000 AND addr = 0x401020;

-- Edit: Add block comment (own line above statement)
UPDATE pseudocode SET comment_placement = 'block1', comment = 'vulnerable call'
WHERE func_addr = 0x401000 AND addr = 0x401020;

-- Edit: Delete comments at a resolved unique anchor
-- Warning: comment = NULL currently clears all placements at that addr.
UPDATE pseudocode SET comment = NULL
WHERE func_addr = 0x401000 AND addr = 0x401020;

-- Read edited pseudocode with comments
SELECT addr, line, comment FROM pseudocode WHERE func_addr = 0x401000;
```

### Cleaning Up Orphan Decompiler Comments

If the database contains stale decompiler comments that no longer attach to current pseudocode, use the orphan comment surfaces instead of trying to clear them through `pseudocode`.

Read-first pattern:
```sql
SELECT func_addr, func_name, orphan_count
FROM pseudocode_v_orphan_comment_groups
ORDER BY orphan_count DESC
LIMIT 20;

SELECT addr, comment_placement, orphan_comment
FROM pseudocode_orphan_comments
WHERE func_addr = 0x401000
ORDER BY addr, comment_placement;
```

Delete-only mutation pattern:
```sql
UPDATE pseudocode_orphan_comments
SET orphan_comment = NULL
WHERE func_addr = 0x401000
  AND addr = 0x401020
  AND comment_placement = 'semi';
```

Notes:
- `pseudocode_orphan_comments` is delete-only.
- Non-empty writes are rejected.
- Use the grouped surface for triage, then the precise table for cleanup.
- After triage, switch to `WHERE func_addr = ...` on both surfaces for the fast path.

---

## Function Summary (func-summary)

Use `function summary` and `func-summary` as equivalent intent.

Trigger contract:
- If the user says `function summary` or `func-summary`, add or update exactly one repeatable function comment on `funcs.rpt_comment`.
- If the user says `add function comment` (singular) without line-specific targets, treat it as a repeatable function comment update.
- If the user explicitly asks for a decompiler heading note or pseudocode block comment, use `pseudocode.comment` with a resolved writable anchor instead.
- Only add many line comments when the user explicitly asks for line-by-line/deep annotation.

Default behavior:
- Write the summary to `funcs.rpt_comment`.
- Use `funcs.comment` only when the user explicitly asks for a non-repeatable function comment.
- Do not expand into line-by-line annotation unless the user explicitly asks for deep annotation.
- When a function is being fully annotated, always add/update this summary as the last semantic step.

Length guidance:
- Use one paragraph minimum when applicable.
- For trivial wrappers/thunks, a shorter concise summary is acceptable.
- Capture function role, key inputs/outputs, important state changes, and notable side effects when relevant.

Canonical SQL pattern:

```sql
SELECT addr, name, comment, rpt_comment
FROM funcs
WHERE addr = 0x401000;

UPDATE funcs
SET rpt_comment = 'One-paragraph summary of what the function does, inputs/outputs, and key behavior.'
WHERE addr = 0x401000;
```

Prompt examples:
- `function summary 0x401000`
- `func-summary DriverEntry`
- `func-summary this function`

---

## Editing Disassembly Comments

Read disassembly-level comments from the `comments` table:

```sql
SELECT COALESCE(NULLIF(comment, ''), NULLIF(rpt_comment, '')) AS comment
FROM comments
WHERE addr = 0x401000
LIMIT 1;
```

**Upsert semantics:** for the `names` and `comments` tables, `INSERT` at an address that already has a value **replaces** the existing entry (IDA allows only one name and one comment-slot per EA). `UPDATE` is equivalent for the in-place case. For `names`, IDA's `SN_CHECK` flag also auto-disambiguates global name conflicts (e.g. `foo` may become `foo_0` if the name is already used at a different EA) — read back the row after writing to see what was actually stored.

The `comments` table supports INSERT, UPDATE, and DELETE:

| Table | INSERT | UPDATE columns | DELETE |
|-------|--------|---------------|--------|
| `comments` | Yes | `comment`, `rpt_comment` | Yes |

```sql
INSERT INTO comments(addr, comment) VALUES (0x401000, 'regular comment');
INSERT INTO comments(addr, rpt_comment) VALUES (0x401000, 'repeatable comment');
UPDATE comments SET comment = 'updated comment' WHERE addr = 0x401000;
DELETE FROM comments WHERE addr = 0x401000;
```

Notes:
- `comments` edits address comments via `set_cmt()`.
- `funcs.comment` / `funcs.rpt_comment` edit true function comments via `set_func_cmt()`.

---

## bookmarks

The `bookmarks` table supports full CRUD for editing marked positions and folder organization:

| Column | Type | Description |
|--------|------|-------------|
| `slot` | INT | Bookmark slot index |
| `addr` | INT | Bookmarked address |
| `description` | TEXT | Bookmark description |
| `inode` | INT | Read-only dirtree inode |
| `folder_path` | TEXT | Writable bookmark folder; `NULL` means root |
| `full_path` | TEXT | Read-only full dirtree path |

```sql
-- List all bookmarks
SELECT printf('0x%X', addr) as addr, description, folder_path FROM bookmarks;

-- Edit: Add bookmark
INSERT INTO bookmarks (addr, description) VALUES (0x401000, 'interesting branch');

-- Edit: Update bookmark description
UPDATE bookmarks SET description = 'confirmed branch' WHERE slot = 0;

-- Edit: Move bookmark into a review folder
UPDATE bookmarks SET folder_path = 'idasql/bookmarks/confirmed' WHERE slot = 0;

-- Edit: Delete bookmark
DELETE FROM bookmarks WHERE slot = 0;
```

For canonical schema and owner mapping, see `../connect/references/schema-catalog.md` (`bookmarks`).

---

## Editing Local Variables (Rename, Retype, Comment)

The `ctree_lvars` table is the editing surface for decompiler local variables. Writable columns: `name`, `type`, `comment`. For full table schema, see `decompiler` skill.

Local-variable edit guidance:
- Inspect/select one deterministic `idx`, then update by `func_addr + idx`.
- Use `UPDATE ctree_lvars SET name = ...`, `type = ...`, or `comment = ...` for local edits.
- For old name-based workflows, first query the candidate rows by `func_addr + name`, choose one `idx`, then update by `idx`.

```sql
-- Inspect current locals before renaming
SELECT idx, name, type, comment
FROM ctree_lvars
WHERE func_addr = 0x401000
ORDER BY idx;

-- Edit: Rename a local variable by index (canonical, deterministic)
UPDATE ctree_lvars SET name = 'buffer_size' WHERE func_addr = 0x401000 AND idx = 2;

-- Edit: Rename by current name after selecting one deterministic idx
UPDATE ctree_lvars SET name = 'buffer_size'
WHERE func_addr = 0x401000
  AND idx = (
    SELECT idx FROM ctree_lvars
    WHERE func_addr = 0x401000 AND name = 'v2'
    ORDER BY idx LIMIT 1
  );

-- Edit: Set local-variable comment by index
UPDATE ctree_lvars SET comment = 'points to decrypted buffer' WHERE func_addr = 0x401000 AND idx = 2;

-- Edit: Change variable type
UPDATE ctree_lvars SET type = 'char *'
WHERE func_addr = 0x401000 AND idx = 2;
```

---

## Editing Decompiler Labels

Read the current labels first, then rename the exact label you observed. Prefer `label_num` identity over guessing from line text.

```sql
-- Inspect labels before renaming
SELECT label_num, name, printf('0x%X', item_addr) AS item_addr
FROM ctree_labels
WHERE func_addr = 0x401000
ORDER BY label_num;

-- Rename deterministically by label number
UPDATE ctree_labels SET name = 'fail' WHERE func_addr = 0x401000 AND label_num = 12;

-- Equivalent UPDATE path
UPDATE ctree_labels
SET name = 'fail'
WHERE func_addr = 0x401000 AND label_num = 12;
```

---

## Editing Types (Create, Modify, Apply)

For type creation, member CRUD, enum values, `parse_decls()`, `applied_types`, and name writes via `names`/`funcs`, see `types` skill.

Quick apply patterns used in annotation workflows:

```sql
-- Apply type to a function
UPDATE funcs SET prototype = 'void __fastcall exec_command(command_t *cmd);'
WHERE addr = 0x140001BD0;

-- Apply/replace the type at any mapped address
INSERT INTO applied_types(addr, decl)
VALUES (0x140001BD0, 'void __fastcall exec_command(command_t *cmd);');
```

---

## Editing Operand Representation (Enum/Struct in Disassembly)

The `instructions` table `operand*_format_spec` columns allow editing operand display:

```sql
-- Edit: Apply enum representation to operand 1
UPDATE instructions
SET operand1_format_spec = 'enum:MY_ENUM'
WHERE addr = 0x401020;

-- Edit: Apply struct-offset representation
UPDATE instructions
SET operand0_format_spec = 'stroff:MY_STRUCT,delta=0'
WHERE addr = 0x401030;

-- Edit: Nested member path uses '/' to separate type names
UPDATE instructions
SET operand0_format_spec = 'stroff:OUTER_T/INNER_T'
WHERE addr = 0x401030;

-- Edit: Clear representation back to plain
UPDATE instructions
SET operand1_format_spec = 'clear'
WHERE addr = 0x401020;
```

All of these work at the disassembly level — no IDAPython needed. The UPDATE is
verified after apply and surfaces a SQL error if the representation didn't take.

The full `operand*_format_spec` vocabulary also covers number bases and other
display forms:

```sql
-- Number base / character
UPDATE instructions SET operand1_format_spec = 'hex'  WHERE addr = 0x401020;  -- also dec/oct/bin
UPDATE instructions SET operand1_format_spec = 'char' WHERE addr = 0x401020;

-- Offsets, sizeof, forced operand text
UPDATE instructions SET operand1_format_spec = 'offset:tbl_start' WHERE addr = 0x401020;
UPDATE instructions SET operand1_format_spec = 'sizeof:MY_STRUCT' WHERE addr = 0x401020;
UPDATE instructions SET operand1_format_spec = 'forced:5 shl 3'   WHERE addr = 0x401020;

-- Sign / bitwise-not modifiers (combine with a base, or use alone)
UPDATE instructions SET operand1_format_spec = 'dec,signed' WHERE addr = 0x401020;
```

Other kinds: `float`, `segment`, `stkvar`, and the `,unsigned` / `,bnot` /
`,nobnot` modifiers. See the `disassembly` skill for the full table.

---

## Editing Union Selection in Decompiled Code

For union selection helpers (`set_union_selection*`, `get_union_selection*`), see `decompiler` skill.

---

## Editing Number Format (Enum Rendering in Decompiled Code)

**Recommended:** Retype the variable to an enum type — IDA's decompiler will then automatically render all constants using enum names:

```sql
-- 1. Define the enum type (skip if it already exists)
SELECT parse_decls('typedef enum { DLL_PROCESS_DETACH=0, DLL_PROCESS_ATTACH=1 } fdw_reason_t;');

-- 2. Retype the parameter/variable
UPDATE ctree_lvars SET type = 'fdw_reason_t'
WHERE func_addr = 0x180001050 AND idx = 1;

-- 3. Verify
SELECT decompile(0x180001050, 1);
```

For per-operand numform control (`set_numform*`, `get_numform*`), see `decompiler` skill.

---

## Mandatory Mutation Loop (For All Edits)

> Follow the read -> edit -> refresh -> verify cycle defined in `connect` Global Agent Contracts.

---

## Performance Tips for Batch Editing

When editing many functions or annotations, keep these costs in mind:

- **`decompile(addr, 1)` triggers a full re-decompilation** (~50-200ms per function). When editing multiple items in the same function, batch all edits before the refresh:
  ```sql
  -- Good: structural typing first, then refresh, then naming cleanup
  UPDATE ctree_lvars SET type = 'MY_CTX *' WHERE func_addr = 0x401000 AND idx = 0;
  SELECT decompile(0x401000, 1);
  UPDATE ctree_lvars SET name = 'ctx' WHERE func_addr = 0x401000 AND idx = 0;
  UPDATE ctree_lvars SET name = 'size' WHERE func_addr = 0x401000 AND idx = 1;
  SELECT decompile(0x401000, 1);  -- final refresh after cleanup
  ```
- **`pseudocode` comment writes (re)decompile the target function** — each write decompiles the containing function to resolve the comment anchor, persists to IDA's user comments store, then invalidates that function's decompiler cache so the next `decompile(addr)` re-renders. You do not need to call `decompile(addr, 1)` yourself between comment writes, but the write is not free — it costs one decompilation of that function.
- **`ctree_lvars` type changes invalidate the decompiler cache** — after changing a variable type, refresh before you trust `idx`/name-based cleanup. The decompiler may split locals or re-render expressions.
- **`save_database()` can be costly** on large databases. Batch all writes and save once at the end of an annotation campaign, not after each edit.

---

## Additional Resources

- For full cleanup workflows, bulk annotation patterns, and complete worked examples: [references/annotation-workflows.md](references/annotation-workflows.md)

---

## See Also

- `types` — define and modify type dictionary entries; apply them via `applied_types` or `funcs.prototype`.
- `decompiler` — pseudocode comments (`UPDATE pseudocode SET comment = ...`); ctree lvar renames/retypes.
- `disassembly` — rename code labels (`names`), apply instruction-level operand formats.

