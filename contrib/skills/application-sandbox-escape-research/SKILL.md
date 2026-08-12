---
name: application-sandbox-escape-research
description: "Use when hunting new escapes from a latest-stable application sandbox or restricted process, including document readers, office suites, messaging clients, media processors, plugin hosts, renderers, security sandboxes, AppContainer, macOS Seatbelt/XPC, Linux seccomp/namespaces/portals, and brokered desktop applications. Builds a capability map, compromised-child harness, broker/IPC audit, and pr..."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: imported
  upstream: C:\Users\Admin\.agents\skills\application-sandbox-escape-research\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\application-sandbox-escape-research\SKILL.md

# Application Sandbox Escape Research

## Objective

Assume attacker code already executes inside the intended restricted child. Find a distinct bug that grants a capability prohibited by the sandbox: arbitrary host file/device access, stronger process execution, credential access, privileged IPC, persistence, or code execution in a broker/system service.

Do not conflate the initial parser/RCE with the escape.

## Phase 1: Pin the Real Sandbox

Record:

- exact latest stable application and helper/broker versions;
- OS build and security updates;
- child process token/UID, groups, integrity, capabilities, entitlements, namespaces, seccomp/seatbelt/AppContainer profile;
- mitigations and dynamic-code policy;
- inherited handles/file descriptors/ports;
- broker and helper process identities;
- standard enterprise policies and plugins;
- launch flags without developer or no-sandbox options.

Completion criterion: sandbox state is captured from the normal shipped launch path.

## Phase 2: Build the Capability Matrix

From the restricted child, test and record:

| Capability | Directly allowed | Brokered method | Expected restriction | Actual result |
|---|---:|---|---|---|
| filesystem/registry | | | | |
| process/thread | | | | |
| network | | | | |
| devices/GPU/media | | | | |
| IPC/services | | | | |
| credentials/keychain | | | | |
| clipboard/UI/input | | | | |
| shared memory/handles | | | | |
| persistence/update | | | | |

Generate traces of denied operations to discover broker fallbacks. Enumerate reachable named pipes, COM/WinRT, XPC/Mach, D-Bus/portals, sockets, ioctls, and inherited object handles.

Completion criterion: desired escape capability is explicitly denied in the baseline and all legitimate broker routes are identified.

## Phase 3: Compromised-Child Harness

Create a harness that runs through the normal sandbox launcher but lets the researcher issue arbitrary protocol-valid child requests. Preserve:

- authentic process token/profile;
- normal IPC channel establishment and endpoint identity;
- generated message formats and version negotiation;
- resource limits and mitigations;
- broker lifecycle and child crash/restart behavior.

Avoid injecting into or unsandboxing the child; that can change inherited capabilities and invalidate findings.

Completion criterion: harness can replay normal child/broker transactions while remaining measurably restricted.

## Phase 4: Broker and IPC Audit

For each method record:

- caller identity validation;
- parameter and enum ranges;
- object ID ownership and generation;
- path canonicalization and final handle checks;
- handle/FD/port type and rights;
- shared-memory size, offset, mutability, and sealing;
- operation performed before versus after validation;
- callback/cancellation behavior after child death;
- broker impersonation or privilege transitions;
- whether policy lives only in child-side generated code.

High-yield tests:

- call methods in legal but unexpected order;
- reuse stale IDs after object destruction/restart;
- duplicate/reorder/cancel asynchronous requests;
- substitute object types or excess-right handles;
- mutate shared memory after validation;
- race path components or links between check and open;
- request relative operations under attacker-selected roots;
- cross child/profile/session identifiers;
- force fallback and error-recovery paths.

Completion criterion: every candidate identifies the missing broker invariant and gained capability.

## Phase 5: Platform-Specific Boundaries

### Windows

- AppContainer/capability SID and integrity label;
- broker COM/RPC/named pipe ACLs;
- duplicated handles and access masks;
- object namespace/session confusion;
- filesystem reparse/rename after broker validation;
- WinRT capability and packaged-app identity;
- privileged service impersonation.

### macOS

- Seatbelt profile and extensions;
- XPC/Mach service entitlement checks;
- audit token and code-signing identity binding;
- security-scoped bookmarks and file descriptors;
- IOSurface/shared memory and GPU/media services;
- helper tools, launchd services, TCC-mediated resources.

### Linux

- seccomp, namespaces, cgroups, capabilities;
- D-Bus and desktop portal sender identity;
- passed FDs and `/proc` behavior;
- user namespaces, mount namespace, and setuid helpers;
- GPU/media ioctls;
- broker daemon and container runtime sockets.

Completion criterion: exploit does not rely on a platform capability intentionally granted by policy.

## Phase 6: Non-Broker Escape Paths

Inspect:

- kernel drivers reachable from the sandbox;
- GPU/media/codec devices;
- local privileged services not designed as the application's broker;
- updater/crash reporter/telemetry handlers;
- temporary files consumed outside the sandbox;
- extension/plugin hosts with stronger profiles;
- clipboard, drag/drop, printing, accessibility, and shell integration;
- nested document or security-analysis sandboxes.

State whether the bug is application-specific or an OS-wide sandbox escape reachable by many restricted processes.

## Phase 7: Prove the Boundary Crossing

Progress from:

1. query a normally denied benign property;
2. obtain a handle/FD with one unauthorized right;
3. read/write a disposable denied test object;
4. execute a fixed benign helper outside the sandbox;
5. assess code execution and persistence.

Capture pre/post token/profile, object rights, responsible broker/service, and audit logs. Do not use an existing user-writable startup location as the sole proof.

## Phase 8: Variants and Stable Validation

Search:

- sibling broker methods using the same path/handle helper;
- other child process types with related policies;
- platform backends where one OS omitted validation;
- generated client checks absent from receiver code;
- updater/crash/printing helpers sharing IPC;
- old sandbox CVE fixes applied only to one method;
- application forks embedding the same broker library.

Re-run on clean latest stable with normal flags and enterprise policy. Apply the novelty gate separately to the escape root cause.

## Common Pitfalls

1. Launching with debug/no-sandbox flags.
2. Claiming escape because the sandbox intentionally grants a capability.
3. Requiring DLL injection that changes process state.
4. Auditing child stubs instead of broker receiver policy.
5. Treating a kernel bug as application-specific without reachability evidence.
6. Combining initial RCE and escape into one unexplained chain.
7. Ignoring inherited handles and startup state.
8. Proving only write to an already user-writable location.

## Verification Checklist

- [ ] Latest stable normal launch path pinned
- [ ] Child sandbox token/profile and inherited objects captured
- [ ] Baseline capability matrix complete
- [ ] Compromised-child harness remains sandboxed
- [ ] Broker method/identity/handle policy mapped
- [ ] Missing invariant and unauthorized capability proven
- [ ] Platform policy confirms capability was not intended
- [ ] Initial code execution and escape root causes separated
- [ ] Clean latest-stable reproduction completed
- [ ] Novelty and sibling-method variants checked

