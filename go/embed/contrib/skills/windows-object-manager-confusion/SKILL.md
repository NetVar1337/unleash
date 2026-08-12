---
name: windows-object-manager-confusion
description: "Use when hunting Windows vulnerabilities involving NT Object Manager namespaces, per-session BaseNamedObjects, shadow directories, symbolic-link objects, named sections/events/mutants, registry symbolic links, globalroot paths, or privileged consumers that confuse attacker-created names with trusted kernel objects. Maps namespace resolution, ACLs, token transitions, and name-squatting races int..."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: stealth
  upstream: C:\Users\Admin\.agents\skills\windows-object-manager-confusion\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\windows-object-manager-confusion\SKILL.md

# Windows Object Manager Confusion Research

## Overview

Hunt privileged consumers that create or open named NT objects using paths a weaker actor can precreate, redirect, shadow, or resolve differently across sessions and tokens. For acquisition-focused research, require `zero-day-target-eligibility` to pass on current stable bytes.

Common objects:

- directories and symbolic-link objects;
- sections and shared-memory mappings;
- events, semaphores, mutants, timers, ALPC ports, and named pipes;
- device and volume objects;
- registry keys and registry symbolic links;
- filesystem paths reached through `GLOBALROOT` or an Object Manager link.

The core invariant is:

> A privileged component must bind a trusted name to the intended object, namespace, owner, type, and security descriptor before exposing or using it.

## When to Use

Use when:

- a service or UI broker uses `BaseNamedObjects`, `Global`, `Local`, `Restricted`, or session-specific names;
- an elevated process creates a section/event/pipe after a low-privileged trigger;
- a path crosses `\Sessions\N\BaseNamedObjects`, `\BaseNamedObjects`, `\Device`, `\??`, or `GLOBALROOT`;
- `NtCreateDirectoryObjectEx`, shadow directories, or `NtCreateSymbolicLinkObject` are involved;
- a registry link may redirect security, value, or deletion operations;
- a consumer assumes only SYSTEM can create an object in a directory but resolves the parent through attacker-controlled namespace state;
- the same logical name resolves differently under another session, process device map, or impersonation token.

## Phase 1: Draw the Namespace Graph

Do not flatten paths. Record each namespace node and edge.

```text
process/session DOS map (\??)
  -> device or volume object
session BaseNamedObjects
  -> shadow/fallback directory
  -> symbolic-link object
  -> target object directory
registry path
  -> REG_LINK target
filesystem bridge
  -> \??\GLOBALROOT\...
```

For every node, record:

- full NT path and object type;
- permanent versus handle-lifetime object;
- owner SID, DACL, integrity label, and mandatory policy;
- creator token/session;
- parent directory handle used for relative opens;
- whether resolution follows a link or opens the link itself;
- shadow-directory fallback behavior;
- visibility from SYSTEM, filtered admin, standard user, AppContainer, service, and another session.

Use WinObj/System Informer, NtObjectManager/NtApiDotNet, `NtQueryDirectoryObject`, `NtQueryObject`, debugger traces, and direct native-API micro-tests.

Completion criterion: the graph explains exactly how the privileged consumer reaches the final object from its original name.

## Phase 2: Establish Creation and Squatting Windows

For each named object:

1. capture a clean run;
2. determine who creates the parent directory;
3. determine whether the final call uses open-if, create-new, or open-existing semantics;
4. precreate the expected name with every type you can legally create;
5. vary security descriptor, permanence, case, and lifetime;
6. close and recreate it during the privileged transition;
7. repeat from the same and a different session.

Test type confusion at the namespace level: section versus symbolic link, event versus directory, pipe endpoint versus filesystem-like name, or registry key versus registry link. A type mismatch may fail safely, alter fallback resolution, or expose a second path.

Completion criterion: document which actor can win the name, when, and what the privileged caller does on collision.

## Phase 3: Test Session and Shadow Semantics

High-yield cases:

- session-local object points to a global target;
- `Restricted` or another child directory shadows a trusted parent;
- a shadow directory supplies fallback names from an attacker-owned directory;
- elevated UI process and SYSTEM service use the same textual name from different sessions;
- UAC secure-desktop transitions cause a privileged process to create objects in a user-influenced session namespace;
- a named section is created after the weak actor precreates a symbolic link at its expected name.

Build a matrix:

| Creator | Consumer | Session | Namespace path | Parent owner | Final object type | Result |
|---|---|---:|---|---|---|---|

Completion criterion: reproduce the same trigger in at least two sessions and explain every difference by namespace resolution or ACLs.

## Phase 4: Registry Link and Security-Operation Tests

Registry symbolic links become high impact when a privileged component applies security or lifecycle operations to a name without opening the final key safely.

Test:

- `REG_OPTION_CREATE_LINK` plus `SymbolicLinkValue` targets;
- open-link versus follow-link behavior;
- DACL, owner, label, inheritance, and recursive tree operations;
- delete-tree, rename, notification, and reopen behavior;
- anonymous, impersonated, and primary-token checks;
- per-user hives, `.DEFAULT`, service profiles, and volatile keys;
- consumer callbacks that accept a PID or operation context and then act under a stronger token.

The dangerous pattern is:

```text
privileged component decides key A is safe
  -> attacker makes A a link to key B
  -> privileged component changes B's DACL/owner/label or contents
```

Use `OBJ_OPENLINK`/open-link equivalents in micro-tests to distinguish operations on the link from operations on its target.

Completion criterion: identify the exact registry operation, effective token, and target key object affected after link resolution.

## Phase 5: Named Section and Shared-Memory Tests

For section creation/opening, record:

- backing file or pagefile-backed status;
- section size, protection, attributes, and maximum access;
- creator and consumer handle rights;
- mapping protections and writable views;
- whether trusted code verifies object type, owner, DACL, or creator;
- downstream services/drivers that trust the section by fixed path.

Test separately:

1. attacker precreates a section;
2. attacker precreates a symbolic link where the privileged process expects a section;
3. privileged process follows the link and creates the section in an attacker-selected directory;
4. attacker opens the resulting section and changes shared data;
5. a second privileged consumer opens the section by fixed name.

A section in a SYSTEM-writable directory is not automatically exploitable. Prove a downstream trust consumer and a security-sensitive field.

Completion criterion: show controlled shared bytes crossing a principal boundary or reject the hypothesis.

## Phase 6: Bridge to Filesystem or Service Sinks

Object Manager primitives often become useful only when composed with:

- a privileged file update root supplied through `GLOBALROOT`;
- a relative filesystem open rooted at an attacker-selected object directory;
- a named pipe used to recover the initiating user's session after SYSTEM execution;
- a scheduled task or COM local server that opens a fixed executable/config name;
- a profile service that expands paths through per-session objects;
- a VSS device object selected through a symbolic link.

When the sink is file-based, load `windows-privileged-file-workflows`. Preserve the namespace graph through the bridge; do not collapse it into a Win32 path.

Completion criterion: every chain stage states the object type, namespace, creator, and token.

## Deterministic Synchronization

Prefer:

- object-directory notifications or repeated `NtOpen*` with a bounded timeout for discovery only;
- oplock/directory notifications once resolution reaches a filesystem object;
- task/service state notifications;
- debugger breakpoints on `NtCreate*`/`NtOpen*` native calls;
- explicit events shared between PoC stages.

Avoid infinite busy loops in the final PoC. They obscure the race window and distort scheduling.

Log native status values and object paths at every call.

## Root-Cause Forms

Use one of these forms:

- **Name squatting:** privileged creator does not secure or exclusively create the expected name.
- **Namespace confusion:** same textual name resolves to attacker-selected object under the privileged caller's session/device map.
- **Shadow-directory abuse:** fallback resolution reaches an attacker-controlled parent.
- **Registry-link traversal:** privileged security/content operation follows a link to an unauthorized target.
- **Object-type trust:** downstream consumer trusts a named object without validating type, owner, security, or provenance.
- **Lifetime race:** closing the only trusted handle permits attacker replacement before later reopen.

## Variant Hunting

Search binaries and traces for:

- hard-coded `BaseNamedObjects`, `Global\`, `Local\`, `Restricted`, and `GLOBALROOT` strings;
- calls to `NtCreateDirectoryObjectEx`, `NtCreateSymbolicLinkObject`, `NtCreateSection`, `NtOpenSection`, and named synchronization APIs;
- registry operations on policy, volatile-environment, service-profile, and `.DEFAULT` paths;
- code that creates a named object after UAC, secure-desktop, logon, or session transition;
- relative native opens with a root directory handle;
- fixed names shared between user-mode services and drivers;
- mitigations that validate only the leaf name or only one namespace.

## Common Pitfalls

1. **Path-string reasoning.** Query the actual object and type.
2. **Ignoring handle lifetime.** Many named objects disappear when the last handle closes.
3. **Assuming global visibility.** Test process device map and session namespace separately.
4. **Claiming impact from arbitrary object creation alone.** Identify a consumer that trusts the object.
5. **Conflating registry and Object Manager links.** Their APIs and follow/open-link semantics differ.
6. **Ignoring secure desktop/UAC transitions.** They can create unexpected cross-session consumers.
7. **Busy-loop stabilization.** Replace it with bounded event-driven observation.
8. **Skipping ACL provenance.** Record inherited versus explicit ACEs and mandatory labels.

## Verification Checklist

- [ ] Complete namespace graph recorded
- [ ] Object type, owner, DACL, label, session, and lifetime captured
- [ ] Creation disposition and collision behavior tested
- [ ] Same trigger compared across principals and sessions
- [ ] Link-follow versus open-link behavior established
- [ ] Trusted handle retention or name reopen identified
- [ ] Downstream consumer and sensitive field/operation proven
- [ ] Final PoC uses bounded synchronization
- [ ] Root cause classified independently of the exploit chain
- [ ] Variants searched by native API and namespace pattern

