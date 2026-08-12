---
name: ui-context
description: "Use when the user refers to what is on screen/selected in IDA GUI."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: imported
  upstream: C:\Users\Admin\.agents\skills\ui-context\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\ui-context\SKILL.md

## Primary Query

Use this function to capture active UI state:

```sql
SELECT get_ui_context_json();
```

Capture once per user question, then reuse that snapshot while answering.
Capture again only when the user asks to refresh/re-check, or when a new UI-referential question starts.

---

## Trigger Patterns

Use this skill for prompts like:
- "what am I looking at?"
- "what is on the screen?"
- "what's selected?"
- "look at what I'm doing"
- references such as "this", "here", "current", "selected", "that", "previous"

---

## Capture Policy

1. Call `get_ui_context_json()` before answering UI-referential prompts.
2. Extract view info: widget type/title and custom-view state.
3. Extract selection info when present: begin/end and preview text lines.
4. Extract code anchor info when present: address, function, segment.
5. If `has_address: false`, explain limits and do not invent code context.

---

## Widget Fields (`focused_widget` / `main_viewer`)

- `type_id` (int): raw IDA `twidget_type_t` numeric id.
- `type_name` (string): the SDK macro name for the widget on the building SDK (for example `"BWN_PSEUDOCODE"`, `"BWN_TILIST"`). The name at SDK slot 10 differs by IDA version (`BWN_TILVIEW` / `BWN_TICSR` / `BWN_TITREE`).
- `canonical_name` (string): a **stable cross-version** name. For the "Local Types" widget (slot 10) this always reads `"BWN_LOCAL_TYPES"`; for every other widget it equals `type_name`. Prefer this for cross-version reasoning.
- `category` (string): high-level grouping — one of `disassembly`, `decompiler`, `hex_view`, `type_view`, `chooser`, `debugger`, `navigation`, `output`, `script`, `auxiliary`, `unknown`.
- `is_address` (bool): widget exposes a meaningful address anchor (true for DISASM, HEXVIEW, PSEUDOCODE, MICROCODE).
- `is_custom_view` (bool): viewer-style widget with custom line content.
- `is_chooser_like` (bool): row-based list/chooser widget where row selection is meaningful.
- `title` (string): the widget's title as IDA reports it.

`main_viewer` carries the same widget fields except `is_chooser_like` (the main viewer is always a viewer, not a chooser).

---

## Temporal Reference Rules

- `this` / `here` / `current` / `selected`: capture a fresh snapshot for this question.
- `that` / `previous` / `earlier`: reuse the most recent snapshot in the same working flow.
- If the user says "refresh", capture a new snapshot immediately.

---

## Runtime Availability and Fallback

- `get_ui_context_json()` is registered in every runtime, but only the IDA GUI
  plugin returns live UI state.
- The live GUI envelope carries `available:true`; the idalib/CLI stub carries
  `available:false` / `capture.source:"cli"` (no error, just no UI). So `available`
  is the reliable GUI-vs-CLI discriminator in both directions — key on it.
- When you get the CLI stub, state this clearly and continue with non-UI queries (explicit addresses/symbols, or DB-orientation queries).

`binary` is database metadata only; it is not a UI context replacement.

---

## Response Template

Use this fixed shape after reading UI context:

- `What You Are Viewing`: widget/view summary.
- `What Is Selected`: selection range and preview (or "no active selection").
- `Code Context`: address/function/segment if available; otherwise mention non-address view.
- `Limits`: runtime constraints or missing fields affecting certainty.
- `Suggested Next Query`: one concrete follow-up command/query.

---

## Examples

### 1) "What am I looking at?"

```sql
SELECT get_ui_context_json();
```

Answer with the template fields above. Include focused widget title/type and current anchor address if present.

### 2) "Look at what I'm doing"

```sql
SELECT get_ui_context_json();
```

Summarize active view and selection first, then propose the next action tied to that context (for example, decompile current function).

### 3) "What's selected right now?"

```sql
SELECT get_ui_context_json();
```

Prioritize selection begin/end and preview text. Explicitly state when there is no active selection.

### 4) "Explain this function"

1. Capture UI context.
2. If function/address context exists, pivot to decompiler.

```sql
SELECT get_ui_context_json();
SELECT decompile(0x401000);
```

Replace `0x401000` with the captured function/address anchor.

### 5) Follow-up: "What about that function?"

Reuse the most recent snapshot from the same flow (do not auto-refresh unless asked), then continue analysis from that stored anchor.

### 6) Non-address view (for example Local Types chooser)

```sql
SELECT get_ui_context_json();
```

If `has_address: false`, report chooser selection details and state there is no code address anchor in this view.

### 7) Plugin API unavailable (CLI/idalib)

If `get_ui_context_json()` cannot run:

- state that UI context is unavailable in this runtime
- ask for explicit symbol/address or continue with non-UI alternatives (for example `SELECT * FROM binary` for DB orientation)

---

## Guardrails

- Do not treat `binary` output as UI context.
- Do not fabricate selection/address fields when absent.
- Do not recapture repeatedly in one answer unless user asks for refresh.

