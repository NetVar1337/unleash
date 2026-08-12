---
name: windows-0day-hunting
description: "Use when hunting new Windows privilege-escalation or security-boundary vulnerabilities in first-party services, scheduled tasks, recovery flows, security products, profile handling, or other privileged workflows. Drives evidence-first attack-surface mapping, state-machine analysis, controlled experiments, exploit-chain construction, variant hunting, and reproducibility testing."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: stealth
  upstream: C:\Users\Admin\.agents\skills\windows-0day-hunting\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\windows-0day-hunting\SKILL.md

# Windows 0-Day Hunting

## Overview

Hunt composition bugs in Windows workflows rather than starting from a favorite exploit primitive. The recurring shape is:

```text
attacker-controlled state
  -> privileged component interprets it
  -> identity/path/state changes between validation and use
  -> privileged write, load, mount, execution, or disclosure
```

Prioritize first-party components that cross user-to-SYSTEM, user-to-other-user, offline-to-online, boot-to-recovery, or encrypted-to-unlocked boundaries. Use the specialist skills when the hypothesis centers on files, profiles/hives, or recovery state.

The methodology was distilled from public MSNightmare/NightmareEclipse repositories and Project Nightcrawler posts. Read [references/msnightmare-method.md](references/msnightmare-method.md) when provenance, examples, or confidence limits matter.

When research is governed by a latest-stable, widespread-deployment, non-public acquisition policy, run `zero-day-target-eligibility` before exploit engineering and again before reporting.

## When to Use

Use for:

- Windows local privilege escalation and sandbox or security-boundary escape research;
- privileged services, COM/RPC servers, scheduled tasks, maintenance jobs, AV/EDR, setup, update, backup, restore, and recovery workflows;
- logic flaws involving path trust, object identity, stale authorization, state transitions, impersonation, or cross-user data;
- turning a suspicious Procmon/ETW/WinDbg observation into a controlled vulnerability hypothesis;
- variant analysis after a Windows LPE or boundary-bypass fix.

Do not use this as the primary workflow for kernel memory corruption, browser exploitation, or network-parser fuzzing. Route those to the corresponding specialist skill.

## Core Research Invariant

Never begin with “how can I use a junction?” Begin with:

> Which privileged action depends on state a weaker actor can influence, and what identity or authorization invariant must remain true from input acceptance through the final side effect?

The bug is the broken invariant. A junction, oplock, hive edit, boot artifact, RPC trigger, or scheduled task is only a way to demonstrate it.

## Phase 1: Build the Research Boundary

Record before testing:

- exact Windows edition, architecture, build, cumulative update, component/package version, and security-product platform/signature version;
- initial token, integrity level, privileges, session, user SID, groups, and whether the actor is local, remote, interactive, service, or recovery environment;
- expected target boundary: SYSTEM, administrator, another user, protected path, encrypted volume, or trusted boot state;
- VM snapshot identifier and rollback procedure;
- one success signal and one negative-control signal.

Completion criterion: another researcher can recreate the starting state without relying on your machine history.

## Phase 2: Select Workflow-Rich Targets

Rank candidates by boundary crossings and statefulness, not merely by binary size.

| Candidate | High-value behavior |
|---|---|
| Security products | quarantine, remediation, scanning, shadow copies, offline scan, exclusions |
| Profile and identity services | logon, profile load/unload, hive mount, known-folder expansion, impersonation |
| Scheduled maintenance | trusted task definition, SYSTEM execution, writable intermediate artifacts |
| Backup/restore/update | snapshots, staging directories, replace/rename, rollback, package extraction |
| Setup and recovery | pre-auth parsing, auto-unlock, unattend/config discovery, offline servicing |
| Virtual disk and storage | mount namespaces, device paths, removable/ISO assumptions, reparse handling |
| Error reporting/diagnostics | collection, staging, compression, trusted helper launch |

Prefer workflows with at least three of these properties:

1. low-privileged trigger;
2. privileged asynchronous worker;
3. attacker-visible intermediate artifact;
4. path reconstructed more than once;
5. temporary directory or snapshot;
6. impersonation transition;
7. persisted state consumed after reboot/logon;
8. final write/load/execute under stronger authority.

Completion criterion: maintain a ranked target list with a concrete trigger and expected privileged side effect for each item.

## Phase 3: Trace One End-to-End Transaction

Use Procmon, ETW/WPR, service logs, WinDbg, API Monitor, RPC/COM tracing, and static RE as appropriate. Capture one clean transaction from low-privileged trigger to privileged sink.

For every meaningful event, record:

- timestamp and thread/process identity;
- token and impersonation state;
- path in Win32, NT, device, and Object Manager form when applicable;
- whether the component holds a handle or later reopens by name;
- requested access, share mode, create disposition, and reparse behavior;
- registry/config value supplying the path or state;
- side effect and cleanup behavior.

Mark all transitions:

```text
validate -> queue -> impersonate/revert -> resolve -> open -> mutate -> execute/load
```

Completion criterion: the trace identifies both the first privileged interpretation of attacker-influenced state and the final security-sensitive operation.

## Phase 4: Build the Hypothesis Ledger

One row per hypothesis:

| ID | Weak actor controls | Privileged consumer | Required invariant | Perturbation | Sink | Observable | Status |
|---|---|---|---|---|---|---|---|

Useful invariant families:

- **Object identity:** the object validated is the object later used.
- **Principal binding:** data belonging to user A cannot select user B's object.
- **Namespace binding:** a path cannot resolve differently across process, session, device, or boot namespaces.
- **Authorization continuity:** access is checked under the correct principal at the final operation.
- **State authenticity:** persisted mode/config accurately represents the current authorized transition.
- **Volume trust:** writable pre-auth media cannot supply trusted commands or configuration.
- **Lifecycle ordering:** cleanup, unload, rename, and replacement cannot race the privileged consumer.

Reject rows that name only a primitive and no invariant.

Completion criterion: every active hypothesis predicts a specific event that distinguishes vulnerable from safe behavior.

## Phase 5: Run Small Differential Experiments

Change one variable at a time:

- path spelling, namespace, case, trailing separators, short name, UNC/device form;
- symlink/junction/reparse presence before validation versus after validation;
- rename/delete/replace while a handle remains open;
- alternate stream versus default stream;
- same artifact under another user, session, volume, or boot phase;
- clean state versus prior workflow state;
- standard user versus filtered admin versus SYSTEM;
- desktop versus Server and stable versus Insider/Canary build;
- cold boot versus reboot, logon, profile unload, recovery entry, or task retry.

Use event-driven synchronization whenever possible. Sleeps may explore timing but cannot establish the final PoC.

Keep a negative control that preserves the expected invariant. If both test and control succeed, the signal is not diagnostic.

Completion criterion: the ledger contains repeatable evidence for or against each hypothesis, not just screenshots or a one-off success.

## Phase 6: Assemble the Minimum Chain

Separate the chain into named stages:

1. **Seed:** create or alter attacker-controlled state.
2. **Trigger:** invoke the legitimate privileged workflow.
3. **Gate:** pause at a deterministic observable event.
4. **Swap:** change object, path, namespace, identity, or state.
5. **Sink:** allow the privileged operation to continue.
6. **Proof:** produce a harmless, unambiguous boundary-crossing result.
7. **Restore:** revert modified system/user state.

First prove the primitive with a benign marker. Then prove impact. Do not add payload complexity before the root cause is stable.

Completion criterion: each stage has an independent log line or trace event and a defined failure reason.

## Phase 7: Stabilize and Measure

For races and asynchronous workflows:

- replace polling and sleeps with oplocks, directory notifications, ETW events, debugger breakpoints, named events, or handle-state observations;
- measure gate-to-swap and swap-to-sink timing;
- run at least 30 trials per relevant build/state;
- record success, timeout, wrong-object, access-denied, cleanup failure, and crash separately;
- test under low/high CPU and I/O pressure only after the baseline is understood;
- ensure cleanup is idempotent after partial failure.

A PoC that works once is a lead. A vulnerability claim requires a reproducible transaction and a root-cause explanation.

Completion criterion: report trial counts and classified outcomes, including failed configurations.

## Phase 8: Root Cause and Variant Hunt

State root cause in implementation terms:

```text
<Component> accepts <attacker-influenced state> under <principal/phase>,
then later <re-resolves/reuses/trusts> it under <stronger authority>
without preserving <identity/authorization/state invariant>, enabling <impact>.
```

Then search siblings:

- callers of the same helper or RPC method;
- other consumers of the same registry/config value;
- similar scheduled tasks and service methods;
- all workflows using the same staging-root convention;
- server/client and online/offline implementations;
- adjacent setup passes, recovery modes, profile types, and volume types;
- pre-fix and post-fix binaries to identify incomplete checks.

Completion criterion: produce a variant matrix with tested status, not a list of guesses.

## Required Research Output

Create these artifacts:

1. environment and build matrix;
2. transaction trace or event timeline;
3. hypothesis ledger;
4. root-cause statement;
5. minimized PoC with stage logging and cleanup;
6. reliability table;
7. affected/unaffected version evidence;
8. variant matrix;
9. remediation invariant and regression-test idea.

## Common Pitfalls

1. **Primitive-first tunnel vision.** A reparse point is not a finding. Identify the violated invariant.
2. **Procmon-only conclusions.** Procmon reveals behavior but not always token state, handle reuse, or internal checks. Corroborate with ETW, debugger, or RE.
3. **Name/handle confusion.** Explicitly determine whether the component reopens by path or acts on the validated handle.
4. **Dirty-state false positives.** Roll back and test clean versus historically primed machines.
5. **Uncontrolled races.** Replace sleeps with observable gates before evaluating impact.
6. **Build overgeneralization.** Component platform updates can matter independently of the OS cumulative update.
7. **PoC complexity hiding root cause.** Reduce to one broken invariant and one harmless proof first.
8. **Missing rollback.** Profile, hive, recovery, and security-product experiments can persist across reboot; document restoration before the first run.

## Verification Checklist

- [ ] Exact build, component version, token, session, and starting state recorded
- [ ] Privileged transaction traced end to end
- [ ] Broken invariant identified independently of the exploit primitive
- [ ] Negative control behaves differently from the vulnerable case
- [ ] Synchronization is event-driven or its limitation is quantified
- [ ] PoC stages and cleanup are independently observable
- [ ] Reliability measured across repeated trials
- [ ] Clean-snapshot reproduction completed
- [ ] Affected/unaffected matrix includes meaningful state and edition differences
- [ ] Variant hunt follows the root-cause pattern
- [ ] Fix recommendation preserves identity/authorization through the final side effect

