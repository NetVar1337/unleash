---
name: advanced-packer-unpacking
description: "Use on authorized packed binaries (Themida/VMProtect/Enigma/etc.) to recover OEP, imports, image."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: re
  upstream: C:\Users\Admin\.agents\skills\advanced-packer-unpacking\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\advanced-packer-unpacking\SKILL.md

# Advanced Packer Unpacking

## Overview
Unpacking is loader reconstruction, not “dump memory and hope.” The deliverable is either a faithful runtime-image artifact or a documented analysis database that preserves sections, imports, relocations, exception metadata, TLS, control-flow metadata, and the transition into original code. Commercial protectors vary by version, options, architecture, and per-build randomization; product names are starting hypotheses, never signatures of one fixed algorithm.

Use separate tracks for the outer packer, code mutation, virtualization, and licensing/runtime checks. Removing the outer loader does not devirtualize protected functions.

## Authorization Boundary
Use only for software the user owns, licensed research samples, CTFs, vendor-approved testing, or isolated malware research. For Apex Legends, League of Legends, or other online games, restrict work to offline, lawfully obtained artifacts and defensive interoperability/research. Do not disable anti-cheat, attach to live game/service processes, evade integrity checks, extract credentials or secrets, automate unfair play, or distribute reconstructed proprietary code.

Work on a copy in an isolated VM. Executing an untrusted sample, changing process memory, forcing control flow, or writing a rebuilt PE requires explicit user direction.

## Success Contract
Choose one before running the target:
1. **Analysis success:** original/native regions and behaviors are inspectable even if no standalone file is emitted.
2. **Runtime-image success:** captured image reproduces mapped code/data with provenance.
3. **Rebuilt-file success:** a new file loads and reaches selected behavior without the original unpacking stub.
4. **Partial success:** one layer is removed and residual mutation/virtualization is mapped.

Do not claim “unpacked” when only one section was dumped or an OEP guess was reached.

## Phase 1 — Immutable Baseline
1. Record authorization, original path, size, SHA-256, signer, format, architecture, subsystem, image base, entry RVA, timestamps, and mitigations.
2. Capture sections, raw/virtual sizes, permissions, entropy, imports/delay imports, relocations, TLS callbacks, resources, overlay, load config, exception/unwind data, exports, and debug metadata.
3. Identify packer clues as evidence: executable writable sections, tiny import table, high-entropy regions, overlay, unusual TLS, loader/API concentration, or late executable writes. None is proof alone.
4. Preserve the original and create a disposable analysis copy/snapshot.

Done when every later runtime mapping can be related to an original file range or marked runtime-created.

### Mapped-image snapshots
A PE-looking dump may use `file offset == RVA` while retaining stale disk `PointerToRawData` fields. Before trusting imports, certificates, sections, or entry bytes, test both disk and memory mappings. Strong mapped-image signals include `file size == SizeOfImage`, an ASLR-loaded `ImageBase`, resolved IAT/TLS pointers, absent Authenticode data, and runtime-populated writable state.

For a mapped snapshot:
1. Parse RVA directories using memory layout without patching the sample.
2. Treat the security directory as a disk-only file offset unless separately captured.
3. Page-map every section and quantify zero/missing pages before analysis.
4. Correlate a same-build disk image by PDB GUID/age, timestamp, section geometry, exports, resources, and embedded build identity.
5. Keep protected disk bytes and runtime-materialized bytes as separate evidence; do not fill absent pages by assumption.
6. Redact tokens, OAuth callback values, usernames, and other transient process state from generated artifacts.

For the complete discrimination, correlation, TLS attribution, hybrid-preloader, and privacy workflow, read `references/mapped-pe-runtime-snapshots.md`.

## Phase 2 — Loader Lifecycle Map
Map transitions rather than single-step everything:
- process creation and earliest executable entry
- TLS callbacks and loader notifications
- memory allocation/reservation and protection changes
- writes into executable or soon-to-be-executable pages
- decompression/decryption loops and source/destination ranges
- import resolution and API indirection
- exception/VEH/SEH-driven transfers
- thread creation/APCs/fibers and alternate entry paths
- transfers from loader-owned code into stable recovered code

For each transition record current VA, module/RVA, page backing, old/new protection, writing instruction, first execution, and run/input. A debugger-visible transfer alone is not proof of OEP.

## OEP and Stable-Code Criteria
Treat a candidate as original-entry-like only when several signals agree:
- control moves from loader/stub into a stable executable region
- compiler/runtime initialization or coherent application CFG appears
- stack/ABI state is plausible
- imports or API wrappers resolve into sustained application behavior
- subsequent execution does not immediately return to a dominant unpack loop
- the region reproduces across controlled runs after rebasing

Many protected binaries have no single classical OEP. Report a set of native islands, virtual-machine entries, and initialization boundaries if that is what the evidence supports.

## Capture Strategy
1. Snapshot after the target region is materialized and before it is destroyed/re-encrypted.
2. Capture all relevant mapped pages plus mappings, protections, module bases, thread contexts, and code-version timestamps.
3. Preserve original and runtime bytes separately; self-modifying pages need versioned captures.
4. Record API/import resolution state and any protector-owned dispatch tables needed for analysis.
5. Hash every artifact and tie it to the exact run and breakpoint/event.

Prefer debugger or DBI read APIs. Do not inject a dumper into a protected live service.

## PE Reconstruction Checklist
A standalone PE may require more than copying mapped bytes:
- DOS/NT/optional headers and architecture consistency
- section RVA/raw layout, alignment, sizes, and characteristics
- entry RVA chosen from evidence
- imports, descriptors, thunks, names, ordinals, and delay imports
- base relocations when rebasing is expected
- TLS directory, callbacks, and index/data semantics
- x64 `.pdata`/unwind information
- load-config data including CFG/CET-related tables where applicable
- resources, exports, runtime data, and required overlay/config data
- correct raw-versus-virtual zero filling

If a field cannot be reconstructed, document the runtime dependency rather than inventing data.

## Import Recovery
Correlate at least two sources:
1. runtime resolver calls or loader notifications
2. final IAT-like memory contents and call sites
3. module export tables/API-set resolution
4. static wrapper/thunk data flow

Distinguish direct imports, delay imports, dynamically resolved APIs, forwarded exports, syscalls, and protector wrappers. Preserve module/name/ordinal and call-site evidence. An address inside a DLL is not enough because versions and forwarding change.

## Anti-Analysis Handling
Detect and classify timing, debugger queries, exception tricks, guard pages, hardware-breakpoint checks, integrity checks, thread hiding, self-debugging, environment checks, and code mutation. First reproduce the divergence between controlled runs. Prefer observation below or beside the checked layer—record/replay, DBI, emulator, hypervisor, or hardware trace—over a pile of blind patches.

Any approved bypass experiment must be minimal, reversible, and confined to the analysis copy. Record original bytes/state, exact changed predicate/API result, and whether behavior changed. A bypass that changes application semantics invalidates later equivalence claims.

## Layered Protector Workflow
For each layer produce:
- layer boundaries and code/data ranges
- materialization/destruction events
- control transfers into the next layer
- persistent runtime services still required
- artifact hash and residual protection

Then route:
- VM dispatcher/bytecode: `virtualization-deobfuscation`
- opaque predicates/CFF/MBA: `binary-obfuscation-deconstruction`
- large runtime traces: `trace-guided-deobfuscation`

## Validation Ladder
1. **Structural:** rebuilt file parses; mappings and directory RVAs are valid.
2. **Disassembly:** recovered executable ranges decode coherently from proven boundaries.
3. **Loader:** isolated launch reaches selected boundary without the original stub, if standalone reconstruction is the goal.
4. **Behavioral:** deterministic fixtures match selected outputs, side effects, exceptions, and API traces.
5. **Differential:** compare original protected run and reconstructed artifact over multiple inputs.
6. **Negative:** malformed/edge inputs preserve relevant failure behavior.

A passing parser is not runtime validation. A single happy-path run is not semantic equivalence.

## Evidence Report
Include protector hypothesis and version confidence; target hash/build; tool versions; original and runtime mappings; OEP/native-island evidence; dump event; import provenance; reconstructed directories; residual VM/mutation; artifact hashes; exact validation runs; and unresolved dependencies.

## Common Pitfalls
1. Calling the first executable write the OEP.
2. Dumping only one region while TLS, unwind, imports, or runtime data live elsewhere.
3. Rebuilding imports from addresses without module/export/forwarder provenance.
4. Assuming Themida/VMProtect versions share handlers or loader structure.
5. Patching every anti-debug check and corrupting target semantics.
6. Ignoring self-modifying code versions.
7. Testing reconstructed files on the production host or online service.
8. Declaring success when virtualization remains untouched.

## Verification Checklist
- [ ] authorization, isolation, original hash, and analysis copy recorded
- [ ] success contract selected
- [ ] static loader metadata and runtime mappings captured
- [ ] OEP/native boundaries supported by multiple signals
- [ ] code versions and executable writes tracked
- [ ] imports and PE directories reconstructed from evidence
- [ ] residual mutation/virtualization explicitly mapped
- [ ] rebuilt artifact validated structurally and behaviorally
- [ ] no live anti-cheat/service bypass or unrequested mutation occurred

