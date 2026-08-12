---
name: windows-profile-hive-research
description: "Use when hunting Windows user-profile and registry-hive vulnerabilities involving ProfSvc, profile load/unload, CreateProcessWithLogon, RegOpenUserClassesRoot, NTUSER.DAT or UsrClass.dat, known-folder and environment expansion, offline registry editing, cross-user hive mounting, or races between profile state and attacker-controlled paths."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: stealth
  upstream: C:\Users\Admin\.agents\skills\windows-profile-hive-research\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\windows-profile-hive-research\SKILL.md

# Windows Profile and Hive Vulnerability Research

## Overview

Treat profile handling as a privileged distributed state machine. Profile paths, user hives, environment values, known folders, session namespaces, cached handles, and logon tokens can disagree about which user or object is being loaded. For acquisition-focused research, require `zero-day-target-eligibility` to pass on current stable bytes.

Core invariant:

> Every hive and profile artifact loaded for principal A must remain bound to A's authorized profile root and object identity through path expansion, file open, mount, publication, use, and unload.

## When to Use

Use for:

- User Profile Service (`ProfSvc`) load/unload and logon races;
- `CreateProcessWithLogonW`, `LogonUser`, `LoadUserProfile`, and profile-triggered process creation;
- `NTUSER.DAT`, `UsrClass.dat`, user classes root, and per-user COM registration;
- `RegOpenUserClassesRoot`, HKCU, HKU, `.DEFAULT`, service profiles, or cross-user mounts;
- `User Shell Folders`, known folders, environment expansion, and profile path discovery;
- offline hive APIs and replacing a hive before load;
- profile paths bridged through Object Manager links, junctions, or `GLOBALROOT`;
- oplock-gated profile loading.

## Phase 1: Build the Identity Matrix

Record every principal participating in the transaction:

| Role | SID | Token type | Logon type | Session | Profile state | Hive state |
|---|---|---|---|---:|---|---|
| initiating actor | | | | | | |
| credentials supplied | | | | | | |
| profile being loaded | | | | | | |
| target hive owner | | | | | | |
| ProfSvc/service worker | | | | | | |
| final consumer | | | | | | |

Do not use usernames as identities. Record SID, profile-list registry entry, filesystem owner, hive metadata, and mounted HKU name.

Completion criterion: every token and hive in the experiment maps to one SID and one intended profile root.

## Phase 2: Draw the Profile State Machine

At minimum model:

```text
credentials accepted
 -> token created
 -> profile root selected
 -> environment/known folders expanded
 -> NTUSER.DAT opened and mounted
 -> UsrClass.dat discovered/opened/mounted
 -> HKCU/HKCR view published
 -> process starts
 -> references drain
 -> hives unload
 -> cleanup
```

For each transition record:

- caller and effective token;
- registry values read;
- exact path before and after expansion;
- NT path and final file ID;
- handles retained or reopened;
- mount name and destination registry namespace;
- asynchronous callback or service boundary;
- rollback behavior on partial failure.

Use Procmon boot/logon capture, User Profile Service operational logs, ETW, WinDbg, registry callbacks where available, and handle inspection.

Completion criterion: identify the first point where one identity chooses a path or hive later consumed under another identity.

## Phase 3: Inventory Attacker-Controlled Inputs

Test independently:

- `ProfileImagePath` and profile-list metadata;
- `User Shell Folders`, especially `Local AppData` and expandable strings;
- environment variables expanded under user or service token;
- directory DACLs and ownership along every profile path component;
- `NTUSER.DAT` and `UsrClass.dat` file lifecycle and share mode;
- per-user Object Manager directory and symbolic-link visibility;
- registry links in per-user policy or known-folder trees;
- profile aliases, renamed users, deleted/recreated SIDs, mandatory/temporary profiles;
- domain, local, Microsoft-account, service, and default profiles.

Distinguish writable data from trusted selectors. A value is high impact when it selects a privileged path, hive, namespace root, or mount target.

Completion criterion: maintain an input-to-consumer map with the reader's effective token.

## Phase 4: Offline Hive Mutation Experiments

Use an offline-registry API or isolated VM tooling to modify a copied hive without loading it into the live registry.

For each experiment:

1. back up bytes and metadata;
2. record original file ID, owner, DACL, timestamps, and size;
3. edit one value;
4. save to a new file;
5. replace or rename atomically where possible;
6. verify the live service consumes the changed value;
7. restore the original and verify unload.

High-yield values are expandable paths and configuration that a privileged profile transition resolves later.

Avoid assuming file content ownership implies authorization to redirect another principal. The question is whether the privileged consumer rebinds the content to the correct SID and root.

Completion criterion: prove one edited value reaches a specific profile-service operation with trace evidence.

## Phase 5: Cross-User and Cross-Namespace Tests

Vary independently:

- actor A triggering logon for B while selecting artifacts under C;
- target account logged on versus logged off;
- same versus different session;
- profile already loaded versus cold load;
- standard user, filtered admin, administrator, service, and SYSTEM target;
- desktop and Server builds;
- `\BaseNamedObjects`, `\Sessions\N\BaseNamedObjects`, `Restricted`, and `GLOBALROOT` paths;
- real directory, junction, Object Manager link, and shadow-directory-backed path.

Test whether:

- a path expanded under B resolves through namespace state controlled by A;
- ProfSvc opens C's hive while publishing it as B's classes root;
- a mount remains reachable after the initiating token changes;
- hive ownership or embedded identity is ignored;
- the service validates the profile root but later reopens a child by attacker-controlled name.

Completion criterion: any cross-user result must identify all three SIDs and the exact identity mismatch.

## Phase 6: Deterministic Load-Race Testing

Good gates include:

- batch oplock on a copied or replacement hive;
- directory notification for profile staging or rename;
- registry notification for mount publication;
- service event or debugger breakpoint at hive open/load;
- explicit helper process whose logon triggers profile loading.

Race sequence:

```text
prepare alternate profile/hive state
 -> arm gate on expected hive
 -> trigger profile load in helper thread/process
 -> wait for privileged break/open
 -> switch path/link/object while blocked
 -> release
 -> verify mounted hive and SID binding
```

Log `RegOpenUserClassesRoot` or equivalent result only after proving which file object was mounted. A successful open alone does not establish arbitrary-hive control.

Completion criterion: gate-to-mount timing is repeatable and no fixed sleep is required.

## Phase 7: Impact Proofs

Progress in this order:

1. cross-user read-only hive mount;
2. controlled benign value visible in the wrong user's view;
3. arbitrary attacker-supplied hive mount;
4. privileged consumer of a controlled per-user registration/config value;
5. code execution or security-boundary impact.

For each step, preserve the root-cause evidence. Do not let COM hijacking, scheduled-task execution, or another final payload obscure the profile bug.

Completion criterion: impact is linked to the wrong hive/path binding, not merely to pre-existing permissions.

## Cleanup and Recovery

Profile experiments can strand loaded hives or corrupt logon state. Before running:

- snapshot the VM;
- export relevant ProfileList and per-user keys;
- copy original hive bytes and security descriptors;
- define how to terminate helper logons and unload hives;
- ensure cleanup runs after timeout and partial initialization;
- verify no test SID remains logged on and no test hive remains mounted;
- reboot and validate normal logon for affected users.

Completion criterion: repeated runs begin from an equivalent profile/hive state.

## Variant Hunting

Search for:

- all callers of profile-load and user-classes APIs;
- every expandable profile/known-folder value read before a privileged file open;
- sibling services that impersonate a user, capture a path, then revert;
- profile migration, backup, restore, provisioning, default-profile, and temporary-profile flows;
- per-user COM, shell extension, scheduled task, and app registration consumers;
- user hive handling in Server, RDS, Fast User Switching, and service accounts;
- fixes that validate the parent directory but not final hive identity;
- stale profile handles or mount names reused across logon sessions.

## Common Pitfalls

1. **Username-based reasoning.** Use SIDs and actual tokens.
2. **Ignoring already-loaded profiles.** Warm and cold paths differ substantially.
3. **Equating pathname with hive identity.** Capture file ID and mounted hive evidence.
4. **Leaving modified `NTUSER.DAT`.** Always preserve and restore exact bytes and ACLs.
5. **Testing one user only.** Cross-user flaws require at least actor, credentialed user, and target matrices.
6. **Ignoring Object Manager namespaces.** Expandable paths can cross session-specific links.
7. **Payload-first testing.** Prove wrong-hive binding with a marker value first.
8. **Assuming Server matches desktop.** Logon/profile mechanics and available triggers differ.

## Verification Checklist

- [ ] SID/token/session matrix complete
- [ ] Profile load/unload state machine traced
- [ ] Input-to-consumer map records effective token
- [ ] Hive file IDs and mount destinations captured
- [ ] Cold and warm profile states tested
- [ ] Cross-user case names actor, credentialed user, and target SID
- [ ] Race uses a deterministic gate
- [ ] Benign wrong-hive marker proven before payload
- [ ] Cleanup restores hive bytes, ACLs, mounts, and logon state
- [ ] Desktop/Server and relevant account types covered
- [ ] Variants search for the identity-binding failure

