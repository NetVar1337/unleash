---
name: enterprise-server-rce-research
description: "Use when hunting new remote code execution vulnerabilities in the latest stable enterprise server, appliance, middleware, management-plane, gateway, identity, backup, monitoring, messaging, database, or file-processing product. Maps unauthenticated and low-privilege network paths, builds protocol-aware harnesses, traces attacker data to native or interpreter sinks, and validates practical serve..."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: imported
  upstream: C:\Users\Admin\.agents\skills\enterprise-server-rce-research\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\enterprise-server-rce-research\SKILL.md

# Enterprise Server RCE Research

## Priority Model

Prefer, in order:

1. unauthenticated default-listening service;
2. pre-auth parser reachable during handshake, discovery, health check, upload, federation, or authentication;
3. low-privilege tenant/user path reaching privileged backend workers;
4. management agent reachable from enterprise networks;
5. file/message ingestion requiring normal user interaction or routine automation.

Reject targets that are latest-version ineligible, niche without strategic deployment, or reachable only through disabled legacy modules.

## Phase 1: Deployment-Accurate Baseline

Record:

- latest stable product/build, hotfixes, plugins, runtime, OS image, and container digest;
- default installation profile and exposed ports;
- service accounts, containers, namespaces, sandboxing, and outbound access;
- authentication modes and first-run configuration;
- cluster versus standalone differences;
- reverse proxy/load balancer normally deployed in front;
- protocol encryption and test certificates;
- clean snapshot and reset automation.

Test both a clean default and a representative enterprise configuration. Do not weaken security merely to expose a harness unless the weakened mode is separately labeled.

Completion criterion: one command or automation recipe restores the exact target state.

## Phase 2: Enumerate Remote Entry Points

Inventory from network capture, binaries, configs, OpenAPI/IDL/protobuf schemas, route tables, and service registration:

- TCP/UDP/QUIC listeners;
- HTTP routes, WebSocket upgrades, gRPC methods, GraphQL operations;
- RPC/DCERPC/COM remoting, Java RMI, .NET remoting, custom IPC relays;
- discovery, heartbeat, replication, backup, restore, migration, import/export;
- file upload, archive extraction, document/media conversion, report generation;
- identity federation, SSO metadata, certificate enrollment, directory sync;
- message queues, webhook receivers, mail/calendar ingestion;
- agent/controller and node/cluster protocols;
- admin APIs accidentally sharing pre-auth middleware.

For each entry point record auth state, parser, worker process, privilege, default exposure, and sensitive sink.

Completion criterion: every listener maps to protocol methods and handling binaries/processes.

## Phase 3: Trace Pre-Auth Data Flow

Capture one valid transaction, then trace fields through:

```text
socket -> framing -> decompression/decryption -> parser -> validation
 -> object construction -> dispatch -> filesystem/process/interpreter/native sink
```

High-value transitions:

- length/count/offset arithmetic before authentication;
- compressed-to-expanded size changes;
- deserialization with type selection or callbacks;
- template/expression/query interpreters;
- archive member paths and link handling;
- command construction and helper-process arguments;
- dynamic module/class/plugin loading;
- path-to-handle reopen under service authority;
- request queued to a more privileged worker;
- SSRF reaching local management or metadata endpoints as a chain primitive.

Completion criterion: identify the earliest untrusted field and final operation under the effective server principal.

## Phase 4: Harness Strategy

Choose the narrowest faithful harness:

- in-process parser/API harness for native libraries;
- socket-level replay proxy preserving state and checksums;
- protocol client mutator for stateful services;
- forkserver/snapshot around a long-lived worker;
- container/VM snapshot for appliance-only targets;
- differential harness across versions or implementations.

Seed from valid production-like transactions. Preserve dependent fields with a grammar or custom mutator. Split campaigns by parser stage so authentication failures do not dominate coverage.

Instrumentation:

- ASan/UBSan/MSan where source builds are realistic;
- page heap, Application Verifier, WinDbg, ETW, ProcDump on Windows;
- sanitizers, rr, gdb, eBPF/uprobes, core dumps on Linux;
- coverage through source instrumentation, DynamoRIO/Frida/QEMU, or protocol-state feedback;
- syscall/file/process traces for logic bugs that do not crash.

Completion criterion: valid seeds reach the intended handler and coverage or semantic feedback distinguishes new paths.

## Phase 5: Bug-Class Campaigns

Run separate campaigns for:

### Native memory safety

- frame lengths, nested counts, decompression, integer conversions;
- lifetime across async callbacks, cancellation, timeout, and reconnect;
- allocator mismatch across modules/plugins;
- malformed optional fields and duplicate records;
- race between disconnect and worker completion.

### Injection and interpreter boundaries

- command/argument construction;
- template, expression, query, rule, workflow, and scripting engines;
- unsafe object deserialization and polymorphic type loading;
- server-side include, transform, and report engines;
- environment/config expansion under service accounts.

### File and package processing

- traversal after canonicalization, links, junctions, hard links, ADS;
- archive extraction races and overwrite semantics;
- signed outer package containing unchecked inner content;
- parser chains selected by filename, MIME, magic, or metadata disagreement;
- temporary files later executed or loaded by privileged jobs.

### Authorization/state machine

- pre-auth method reachable after failed/partial negotiation;
- request smuggling between proxy and backend;
- cross-tenant object identifiers;
- stale session/resume tokens;
- controller/agent trust confusion;
- operation validated as user but completed as system worker.

Completion criterion: each campaign has a corpus, feedback signal, timeout policy, and triage queue.

## Phase 6: RCE Triage

For each candidate prove:

1. remote reachability on default/representative deployment;
2. exact authentication and interaction requirement;
3. root cause and affected process;
4. controlled data, offset, type, target, or interpreter expression;
5. mitigations and process boundary;
6. service account privileges and container/sandbox escape needs;
7. network egress and lateral value;
8. clean latest-stable reproduction.

Use harmless proof first: controlled crash, marker file in test directory, predictable callback to a lab listener, or execution of a benign fixed command in an isolated target.

Completion criterion: impact follows from the bug rather than preexisting administrative configuration.

## Phase 7: Cluster and Enterprise Variants

Test:

- node versus controller/coordinator;
- primary versus replica;
- upgrade/migration compatibility endpoints;
- backup/restore and disaster-recovery workers;
- Windows versus Linux packages;
- embedded JRE/.NET/Python/runtime versions;
- appliance and cloud-managed editions;
- direct listener versus standard reverse proxy;
- default and hardened authentication.

A bug in a rarely exposed worker can still be high value if normal enterprise automation feeds it attacker-controlled data.

## Common Pitfalls

1. Fuzzing random bytes before preserving protocol state.
2. Claiming unauthenticated reachability when a proxy or enrollment secret is required.
3. Testing an optional plugin absent from widespread deployments.
4. Confusing SSRF or file write with RCE before proving the chain.
5. Ignoring clean-install defaults and cluster topology.
6. Using only crash feedback for logic and authorization bugs.
7. Failing to update embedded runtimes independently of product version.
8. Testing production/live services instead of a controlled product instance.

## Verification Checklist

- [ ] Eligibility gate passes on latest stable
- [ ] Default and representative enterprise deployments captured
- [ ] Listener/method/process/privilege map complete
- [ ] Valid transaction traced through final sink
- [ ] Stateful harness reaches intended handler
- [ ] Campaigns separated by bug class
- [ ] Authentication, interaction, and default exposure proven
- [ ] Root cause and controlled primitive established
- [ ] Cluster/platform variants tested
- [ ] Novelty search and clean reproduction completed

