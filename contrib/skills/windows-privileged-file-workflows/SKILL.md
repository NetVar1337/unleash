---
name: windows-privileged-file-workflows
description: "Use when researching Windows privileged file-operation vulnerabilities involving services, Defender or other security products, update/remediation pipelines, Cloud Files placeholders, VSS, reparse points, rename/delete semantics, alternate data streams, oplocks, file locks, or name-to-handle races. Builds deterministic path-confusion and file-lifecycle experiments from traces to minimized PoCs."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: stealth
  upstream: C:\Users\Admin\.agents\skills\windows-privileged-file-workflows\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\windows-privileged-file-workflows\SKILL.md

# Windows Privileged File Workflow Research

## Overview

Find broken object-identity and authorization invariants in privileged Windows file workflows. The highest-value pipelines scan, stage, hydrate, quarantine, update, snapshot, remediate, restore, rename, or execute files asynchronously. For acquisition-focused research, require `zero-day-target-eligibility` to pass on current stable bytes.

Model the workflow as handles and objects, not path strings:

```text
attacker name -> initial object -> privileged validation -> queued work
              -> name/namespace/lifecycle change -> privileged reopen
              -> privileged write/read/load/execute
```

A vulnerability exists when the privileged consumer reaches a security-sensitive object that was not the object authorized at the correct boundary.

## When to Use

Use for:

- reparse point, junction, symbolic-link, mount-point, or path normalization hypotheses;
- oplock- or notification-gated races;
- rename-by-handle, delete-pending, replace, supersede, hard-link, and share-mode behavior;
- VSS or other snapshot-backed confusion;
- Cloud Files sync roots, placeholders, hydration/dehydration, and provider callbacks;
- Defender scan, remediation, quarantine, update, signature, or offline-scan file flows;
- alternate data stream handling, especially `Zone.Identifier` and metadata streams;
- virtual disk, ISO, removable-volume, and device-path differences;
- denial or integrity bugs caused by attacker-held file locks.

Do not use for pure Object Manager section/event/mutant confusion without a filesystem leg; use `windows-object-manager-confusion`.

## The Handle Ledger

For every important file operation, record:

| Stage | Process/thread | Token | Requested path | Resolved object | Access | Share | Disposition/options | Handle reused? |
|---|---|---|---|---|---|---|---|---|

Resolve and preserve:

- Win32 path;
- `\\?\` or `\\.\` path;
- `\??\` NT path;
- `\Device\...` path;
- volume GUID and file ID when possible;
- reparse tag and target;
- default stream versus named ADS;
- whether `FILE_OPEN_REPARSE_POINT` or equivalent no-follow behavior is used.

Completion criterion: you can point to the exact transition where the consumer abandons a trusted handle or file ID and resolves an attacker-influenced name again.

## Phase 1: Trace the Legitimate Transaction

Trigger the workflow once without interference. Use Procmon plus at least one corroborating source:

- ETW/WPR for timing and provider events;
- WinDbg breakpoints on `NtCreateFile`, `NtSetInformationFile`, `NtFsControlFile`, `CreateFileW`, Cloud Files APIs, or component-specific routines;
- API Monitor or Frida for user-mode call arguments;
- static RE for path construction, flags, and retry logic;
- Object Manager and handle inspection through WinObj, Process Explorer, System Informer, or NtApiDotNet.

Mark:

1. attacker-controlled root or artifact;
2. privileged first open;
3. validation or classification;
4. asynchronous boundary;
5. reopen, rename, or remediation;
6. final sensitive operation;
7. cleanup.

Completion criterion: capture a timestamped transaction with the responsible process and the exact final sink.

## Phase 2: Enumerate Mutation Primitives

Test primitives independently before chaining them.

### Name redirection

- directory junction/mount point;
- filesystem symbolic link where available;
- Object Manager link feeding a Win32/NT path;
- volume mount point or virtual disk device path;
- relative path under a handle-controlled root;
- drive-letter, volume GUID, UNC, SMB, WebDAV, and local-device variants.

### Identity and lifecycle changes

- rename by handle while retaining the original handle;
- mark delete-pending, then recreate the original name;
- replace/supersede with different share modes;
- swap parent directory after child validation;
- hard-link creation or removal;
- stream creation/deletion while the base file remains stable;
- snapshot creation followed by live-object mutation.

### Availability and timing controls

- batch/exclusive oplock;
- byte-range lock and restrictive share mode;
- `ReadDirectoryChangesW` notification;
- Cloud Files fetch/validate/cancel callback;
- service-status notification;
- ETW event or debugger breakpoint;
- transaction open/rollback where TxF behavior is relevant.

Completion criterion: each primitive has a micro-test proving its exact behavior on the target build and filesystem.

## Phase 3: Design a Deterministic Gate

Choose a gate immediately before the vulnerable decision.

Preferred order:

1. oplock break caused by the privileged open;
2. Cloud Files callback caused by hydration or validation;
3. directory-change event identifying a newly created staging object;
4. ETW event tied to the relevant operation;
5. service or task state transition;
6. debugger breakpoint;
7. timed polling only for reconnaissance.

For an oplock gate, record:

- handle access and share flags;
- requested oplock level or legacy FSCTL;
- expected breaker process and operation;
- whether the break blocks or merely notifies;
- state changed before acknowledging/closing;
- timeout and cancellation behavior.

Do not assume an oplock means the consumer is paused at the desired code path. Verify with a trace or stack.

Completion criterion: gate activation predicts the privileged operation and the swap occurs before its final name resolution or write.

## Phase 4: Test High-Yield Invariants

### Validate-by-name, use-by-name

Test whether the component validates a user-controlled file, queues work, and later reopens the same string after the object or parent changed.

Safe behavior: retain a trusted handle or revalidate final file ID, volume, owner, DACL, reparse state, and destination.

### Trusted staging root

Test temporary and update directories whose names are created by a privileged component but whose contents, namespace root, or parent traversal can be influenced.

Questions:

- Can the attacker predict or observe the directory name?
- Can they hold or replace children?
- Is the source root accepted through a global/session/object namespace?
- Does the service copy trusted output back into a protected destination?

### Snapshot/live-object mismatch

Test whether a privileged component creates or scans VSS data but later uses a live path, attacker-provided name, or link to select snapshot content.

Record snapshot device name and creation time. Distinguish immutable snapshot data from mutable directory entries that refer to it.

### Cloud Files trust inversion

For sync roots and placeholders, vary:

- provider identity and registration scope;
- hydration policy and validation-required modifier;
- placeholder metadata, size, file identity, in-sync state, and cloud tag;
- callback timing and returned data;
- rename/delete/recreate before and after hydration;
- behavior under security-product scanning or remediation.

Ask whether the privileged consumer treats provider-supplied content or metadata as the original trusted file and writes it back under stronger authority.

### ADS size and metadata assumptions

Test base file and streams independently:

- zero/small/huge `Zone.Identifier`;
- delayed or non-terminating remote stream reads;
- malformed or changing stream size;
- local, SMB, WebDAV, and virtual-disk behavior;
- quota and disk-space impact;
- whether size limits applied to the base file are omitted for streams.

Completion criterion: each hypothesis has a vulnerable case and a safe negative control differing in one property.

## Phase 5: Security-Product Workflow Matrix

Security products expose unusual privileged file workflows even when UI protections are disabled. Test the API/workflow itself, not just real-time protection.

| Workflow | Inputs to vary | Sensitive side effect |
|---|---|---|
| Resource scan | scheme, path, source volume, stream, scan flags | privileged open/classification |
| Remediation/quarantine | object lifetime, cloud tag, snapshot, destination | delete/copy/rewrite |
| Signature/platform update | staging root, file locks, update RPC, rollback | trusted engine/data replacement |
| Offline scan | persisted recovery state and config | pre-auth execution, auto-unlock |
| Cloud submission/cache | ADS and remote read behavior | local cache, disk exhaustion, disclosure |

Also test passive mode, disabled real-time protection, Server, and component-platform version independently. A workflow can remain reachable when the normal monitoring feature appears off.

Completion criterion: report reachability and result for each relevant mode rather than assuming the UI state governs the backend.

## Phase 6: Minimize the PoC

Structure the PoC as:

```text
prepare -> trace checkpoint -> arm gate -> trigger -> swap -> release -> prove -> restore
```

Requirements:

- random per-run working directory;
- no fixed sleeps in the final synchronization path;
- explicit timeouts;
- log paths, file IDs, volume IDs, reparse targets, and NTSTATUS/Win32 errors;
- harmless proof first, such as a marker in a controlled protected test target;
- cleanup that tolerates missing, moved, delete-pending, or still-open objects;
- no giant embedded payload unless the payload itself is under test.

For reliability, classify at least 30 trials:

- success;
- gate timeout;
- privileged consumer never reached artifact;
- swap too early/late;
- wrong object opened;
- access denied;
- object remained locked;
- cleanup failure.

Completion criterion: a failed run explains its phase and leaves the VM recoverable.

## Variant Hunting

Once root cause is known, search for:

- same file helper or path-normalization routine in sibling services;
- all RPC/COM methods accepting paths or update roots;
- every Cloud Files callback or placeholder state consumer;
- all security-product handlers for base file versus ADS;
- every scheduled task consuming the component's output;
- live versus VSS versions of the same workflow;
- local versus SMB/WebDAV/virtual-disk sources;
- desktop versus Server and online versus recovery implementations;
- pre- and post-patch callers that received only partial mitigations.

Search for the missing invariant, not exploit-specific names.

## Common Pitfalls

1. **Treating a pathname as an object identity.** Record file ID, volume, and handle lifetime.
2. **Oplock folklore.** Confirm the breaking operation and call stack.
3. **Ignoring share flags.** They often determine whether replace, rename, or lock primitives work.
4. **Assuming reparse checks are recursive.** Test every path component and post-check reopen.
5. **Ignoring ADS.** Metadata streams can bypass size, cache, and remediation assumptions.
6. **Ignoring component updates.** Defender platform/engine behavior can change outside monthly OS patches.
7. **Using resource exhaustion before proving the bug.** First show the omitted bound or lifecycle error with a small controlled case.
8. **Leaving handles behind.** Track every handle and cancel outstanding I/O during cleanup.

## Verification Checklist

- [ ] End-to-end privileged file transaction captured
- [ ] Handle ledger includes path, file ID, volume, access, share, disposition, and reparse behavior
- [ ] Exact identity/authorization invariant stated
- [ ] Mutation primitives validated independently
- [ ] Gate verified against the privileged call stack or trace
- [ ] Negative control included
- [ ] Local/remote, default-stream/ADS, and relevant volume types tested
- [ ] Security-product mode and component versions recorded where relevant
- [ ] PoC uses event-driven synchronization and explicit timeouts
- [ ] Reliability outcomes classified across repeated trials
- [ ] Cleanup survives partial execution
- [ ] Variant search follows the root cause

