---
name: windows-recovery-state-research
description: "Use when hunting Windows pre-authentication and recovery vulnerabilities involving WinRE, WinPE, BitLocker auto-unlock, Defender Offline, ReAgent.xml, BCD, unattended setup, offline servicing, recovery or EFI partitions, removable media, and persisted state that changes trusted behavior across reboot."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: stealth
  upstream: C:\Users\Admin\.agents\skills\windows-recovery-state-research\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\windows-recovery-state-research\SKILL.md

# Windows Recovery-State Vulnerability Research

## Overview

Hunt trust failures that appear only when Windows transitions between the online OS, reboot orchestration, WinRE/WinPE, Defender Offline, setup, and BitLocker unlock states. For acquisition-focused research, require `zero-day-target-eligibility` to pass on current stable bytes.

The key invariant is:

> Pre-authentication code and configuration must come from authenticated, correctly bound storage, and an unlocked OS volume must not become accessible to attacker-controlled recovery code or state.

These bugs are usually state-machine and provenance failures, not cryptographic breaks.

## When to Use

Use for:

- WinRE/WinPE command or configuration discovery;
- BitLocker volume availability before user authentication;
- Defender Offline scan scheduling and persisted recovery state;
- `ReAgent.xml`, BCD entries, recovery sequence, or scheduled operations;
- `unattend.xml`, `RunSynchronous`, Windows PE setup passes, or offline servicing;
- writable recovery/EFI/removable media consumed as trusted input;
- transaction logs, restore artifacts, or filesystem metadata interpreted during recovery;
- behavior that depends on whether a workflow was previously initiated in the full OS.

## Phase 1: Establish the Boot Trust Model

Record:

- hardware/VM, UEFI/legacy mode, Secure Boot, TPM version and state;
- BitLocker conversion/protection status, encryption method, key protectors, auto-unlock state, and PIN/startup-key policy;
- OS edition/build and exact `winre.wim` build/hash;
- recovery partition type, GPT GUID, filesystem, offset, size, ACLs, and mount state;
- EFI and other auxiliary partitions;
- BCD store hash/export and recovery sequence;
- `reagentc /info` output and `ReAgent.xml` hash/content;
- Defender platform version and offline-scan state where relevant;
- physical/removable media assumptions.

Do not claim a BitLocker bypass until the protector model is explicit. TPM-only and TPM+PIN are materially different.

Completion criterion: another researcher can rebuild the same boot and protector state from the record.

## Phase 2: Draw the Cross-Reboot State Machine

Model at least:

```text
online OS
 -> privileged feature schedules recovery/offline action
 -> state/config written to disk/BCD
 -> reboot manager selects WinRE
 -> BitLocker key release/volume unlock
 -> WinRE locates config/artifacts
 -> command/component starts
 -> operation completes or aborts
 -> state is cleared or persists
 -> return to online OS
```

For each transition record:

- component making the decision;
- storage location and writer permissions;
- integrity/authenticity protection, if any;
- volume and path resolution order;
- whether removable or alternate partitions are searched;
- whether OS volume unlock occurs before configuration validation;
- cleanup/persistence behavior after cancellation, failure, and successful completion.

Completion criterion: identify each persisted state field that survives reboot and every parser/launcher that consumes it.

## Phase 3: Inventory Pre-Auth Inputs

Search and diff:

- recovery partition `Recovery\WindowsRE` tree;
- `ReAgent.xml` and sibling files;
- BCD device/path objects and recovery sequence;
- Defender Offline directories and operation parameters;
- Panther/setup/unattend locations recognized by WinPE;
- startup scripts, answer files, drivers, packages, and offline-servicing manifests;
- `System Volume Information`, transaction/restore metadata, and filesystem logs;
- EFI boot applications and configuration;
- removable media roots and volume-label/GUID-based discovery.

For every artifact record:

| Artifact | Writer | Writable pre-auth? | Consumer | Search order | Authenticated? | Sensitive effect |
|---|---|---|---|---|---|---|

Completion criterion: all command-bearing and path-bearing artifacts have provenance and consumer entries.

## Phase 4: Differential State Testing

Change one state variable at a time:

- clean machine versus one where Defender Offline was initiated once;
- operation pending, completed, canceled, interrupted, and stale;
- warm reboot, cold boot, power loss, and recovery entered through different paths;
- internal recovery partition versus removable USB versus copied EFI/recovery contents;
- valid, missing, malformed, duplicated, and conflicting config files;
- OS/recovery image versions matched versus mismatched;
- recovery enabled/disabled, partition relocated, and custom WinRE image;
- TPM-only, TPM+PIN, recovery-password-only, and suspended protection;
- Windows 10/11 and relevant Server builds.

After every run, recapture BCD, `ReAgent.xml`, mount/unlock events, and artifact hashes. Machine history is a test variable, not background noise.

Completion criterion: a table identifies the minimum state transition required for the unexpected behavior.

## Phase 5: Trace File and Component Discovery

Static and dynamic options:

- mount `winre.wim` read-only and inventory binaries, services, scripts, manifests, and strings;
- compare same-named binaries in normal Windows and WinRE by hash, imports, strings, and CFG;
- Procmon in a lab WinPE/WinRE environment where feasible;
- boot logging, ETW, serial/kernel debugging, and filesystem minifilter traces;
- reverse engineer code that builds search paths or parses `ReAgent.xml`, BCD, setup XML, or transaction metadata;
- inspect signature verification and device/volume binding before launch.

When a component exists both online and in WinRE, diff behavior rather than assuming equivalence. Recovery-only code paths are high-value.

Completion criterion: identify the exact parser/launcher and its input search order.

## Phase 6: Test Provenance and Volume Binding

High-yield hypotheses:

### Trusted config from writable media

A recovery component locates XML, logs, scripts, or metadata on a partition an attacker can modify offline, then executes or interprets it after the OS volume is unlocked.

Test which property selects the source: disk number, partition GUID, offset, relative path, volume label, first match, or copied state.

### Stale privileged mode

An online privileged workflow sets “offline scan,” “scheduled operation,” or similar mode. Cancellation or completion fails to clear all state, so later manual WinRE entry inherits stronger behavior.

### Search-order confusion

An artifact on removable or auxiliary media shadows the intended recovery-partition artifact.

### Unattended setup crossover

WinRE enters or invokes a Windows PE/setup path that honors an answer file or synchronous command from writable storage.

### Auto-unlock before trust validation

The OS volume is unlocked for a legitimate recovery operation before the command/config that runs in that environment is authenticated and bound to the trusted recovery image.

Completion criterion: prove the untrusted artifact's bytes and storage provenance reach the pre-auth consumer after unlock.

## Phase 7: Minimize the Proof

Use the smallest harmless proof:

1. create a marker on WinRE RAM drive;
2. print environment and mounted-volume state;
3. demonstrate read access to a non-sensitive marker on the OS volume;
4. demonstrate write access only in a disposable test location;
5. only then assess full boundary impact.

Record:

- exact copy paths and hashes;
- commands used to mount/unmount partitions and WIMs;
- BCD/ReAgent before/after diffs;
- key protector state;
- boot entry method and keys held, if interaction matters;
- video/serial logs with timestamps;
- restoration commands.

Completion criterion: the trigger consists only of the required state and artifacts; unrelated setup-generator content is removed.

## Phase 8: Root Cause and Variants

Root-cause form:

```text
During <recovery state>, <component> accepts <artifact/state> from <untrusted or incorrectly bound source>
after <volume unlock/privileged transition> and performs <command/read/write> without authenticating
<provenance, device binding, or current operation state>.
```

Variant search:

- every `ScheduledOperation` and `OperationParam` consumer;
- all WinRE tools that unlock or mount the OS volume;
- other setup passes and answer-file search locations;
- reset, repair, restore, startup repair, update rollback, and OEM tools;
- same-named online/WinRE binary pairs;
- alternate removable filesystem types and partition placements;
- stale-state paths after interrupted operations;
- recovery image updates that leave old configs or BCD state;
- Server/Core and custom enterprise recovery images.

## Cleanup and Safety of the Test Environment

Before modifying recovery state:

- export BCD and copy original `ReAgent.xml`;
- record partition GUIDs and offsets;
- create full VM/disk snapshots, not only OS checkpoints;
- retain BitLocker recovery material outside the test VM;
- define commands to restore BCD, recovery registration, and partition contents;
- test recovery restoration before destructive experiments;
- never infer clean state from successful normal boot.

Completion criterion: the original recovery sequence and BitLocker state are verified after restoration.

## Common Pitfalls

1. **Calling auto-unlock a cryptographic break.** State the actual trust/provenance failure.
2. **Ignoring machine history.** Prior offline-scan/setup state can be the trigger.
3. **Using a huge generated unattend file.** Minimize to the exact honored setting and command.
4. **Assuming partition location implies trust.** Record who can modify it offline and how the consumer binds to it.
5. **Mixing TPM-only and TPM+PIN conclusions.** Report protector-specific results.
6. **Testing only one recovery-entry path.** Shift-restart, automatic repair, BCD boot, and offline scan can differ.
7. **Ignoring WinRE image version.** Recovery images can lag the online OS.
8. **Failing to restore BCD/ReAgent.** Cross-reboot state can survive the PoC.

## Verification Checklist

- [ ] UEFI/Secure Boot/TPM/BitLocker protector model recorded
- [ ] OS and WinRE versions/hashes recorded
- [ ] Recovery/EFI partition provenance and writeability documented
- [ ] Cross-reboot state machine mapped
- [ ] Pre-auth artifact search order traced
- [ ] Clean versus historically primed states compared
- [ ] Unlock occurs before/after trust validation established
- [ ] Minimal harmless proof reproduced from a full disk snapshot
- [ ] Protector, edition, build, and recovery-entry matrix tested
- [ ] BCD, ReAgent, partitions, and BitLocker state restored and verified
- [ ] Variant search covers sibling recovery operations and parsers

