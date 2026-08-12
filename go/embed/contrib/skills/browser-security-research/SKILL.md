---
name: browser-security-research
description: "Use when hunting new vulnerabilities in the latest stable Chromium, Firefox, WebKit/Safari, or embedded browser stack, including renderer RCE, IPC/broker flaws, sandbox escapes, site-isolation violations, JIT/compiler bugs, DOM/media/font/image parsers, GPU processes, extensions, and browser-to-OS exploit chains."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: security
  upstream: C:\Users\Admin\.agents\skills\browser-security-research\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\browser-security-research\SKILL.md

# Browser Security Research

## Campaign Goal

Prioritize latest-stable bugs that provide:

1. renderer/native code execution from web content;
2. renderer-to-browser/GPU/network/utility process escape;
3. site-isolation or cross-origin data compromise;
4. browser-to-OS sandbox escape;
5. zero/one-click chains in widely deployed embedded webviews.

XSS alone is excluded. A web primitive qualifies only when it crosses a native, origin, process, sandbox, or OS boundary.

## Phase 1: Pin Stable Bytes and Process Model

Record:

- exact latest stable browser and engine build;
- OS build, architecture, hardware features, GPU/driver, and mitigations;
- component-updated modules/codecs/models;
- command line and policy state without unsafe debugging flags;
- process tree, sandbox profiles/tokens, site isolation, JIT mode, and feature flags;
- symbols/source revision corresponding to shipped bytes;
- clean profile and reset procedure.

Do not base eligibility on Canary/Nightly/Technology Preview. Use those channels only for differential clues, then reproduce on stable.

Completion criterion: shipped stable binary maps to an exact source revision or a documented binary-only baseline.

## Phase 2: Select a Boundary

### Renderer attack surfaces

- JavaScript/Wasm JIT and runtime objects;
- DOM/layout/style state machines;
- image, media, font, PDF, archive, and structured-data parsers;
- WebRTC, WebGPU/WebGL, WebCodecs, accessibility, printing;
- extension/content-script bridges;
- lifetime across navigation, frame detach, workers, and async callbacks.

### Sandbox boundaries

- renderer-to-browser IPC;
- GPU, network, audio, storage, PDF, utility, and crash-handler processes;
- brokered filesystem, registry, device, and handle operations;
- shared memory, data pipes, serialized handles, and object capabilities;
- platform service bridges: Windows COM/WinRT, macOS XPC/IOSurface, Linux portals/D-Bus;
- kernel-facing graphics and media interfaces reachable from restricted processes.

Choose one boundary and document attacker preconditions and desired capability.

## Phase 3: Build the Reachability Graph

For web-visible API or compromised-renderer input, trace:

```text
IDL/API -> binding conversion -> renderer implementation -> IPC serialization
 -> receiving process validation -> broker/platform operation -> OS object
```

Record:

- feature/policy/origin gates;
- object ownership and lifetime;
- process performing each validation;
- integer and type conversions;
- associated endpoint/frame/document identity;
- capability/handle rights transferred;
- callback ordering during navigation, destruction, crash, and reconnect.

Completion criterion: every edge identifies the trust level and validation owner.

## Phase 4: Renderer Bug Campaigns

### Lifetime/state

Generate sequences involving:

- create/use/detach/navigate/destroy;
- nested event loops and synchronous callbacks;
- worker termination and page freeze/restore;
- media/GPU reset and context loss;
- cross-thread ownership and cancellation;
- history/BFCache/prerender transitions.

### JIT/compiler

Differentially stress:

- type feedback invalidation;
- bounds-check elimination;
- representation changes and NaN/tagging;
- deoptimization and exception edges;
- Wasm table/memory growth and tier transitions;
- optimizer assumptions across side effects.

Require a minimized semantic mismatch or sanitizer finding before exploit work. Do not classify ordinary spec divergence as memory corruption.

### Parsers

Use structure-aware fuzzing with valid corpora, incremental/streaming input, truncation, metadata disagreement, extreme dimensions/counts, and cross-parser handoff.

Completion criterion: harness reaches real shipped parsing code with sanitizers or reliable crash triage.

## Phase 5: IPC and Sandbox-Escape Campaigns

For every reachable message/method, test:

- enum/range/count and nested object validation;
- endpoint/frame/process identity binding;
- stale object IDs after navigation or process reuse;
- duplicate, reordered, canceled, and late replies;
- handles with excess rights or wrong object types;
- shared-memory size, offset, mutability, and TOCTOU;
- broker path canonicalization and reparse/symlink behavior;
- privileged service methods exposed to a restricted process;
- receiver assumptions enforced only by generated client code;
- feature paths that bypass the normal broker.

Build a compromised-renderer harness where supported. A renderer bug should not be required merely to test broker validation.

Completion criterion: each candidate demonstrates a capability unavailable under the intended sandbox policy.

## Phase 6: Site Isolation and Origin Boundaries

Test identity continuity across:

- redirects, opener relationships, portals/fenced frames/prerender;
- process swaps and speculative frames;
- blob/filesystem/data URLs;
- service/shared workers and storage partitions;
- extension and browser-internal schemes;
- crash recovery and session restore;
- credentialless/cross-origin isolation modes.

Prove actual cross-origin data or authority, not only unexpected process co-location.

## Phase 7: Exploitability and Chains

For memory corruption, determine:

- controlled read/write/free and object shape;
- allocator partition and quarantine behavior;
- pointer compression, CFI, CET/PAC, MTE, MiraclePtr/BackupRefPtr-like defenses;
- JIT RWX/W^X policy;
- process token/profile and available system calls;
- stable primitive across restarts and normal browser flags.

Keep renderer and escape root causes separate. A full chain report should show which primitive crosses each boundary.

## Phase 8: Variant and Stable Validation

- diff recent security fixes and hardening changes;
- search sibling message handlers and platform backends;
- compare browser implementation with embedded WebView/Electron-like consumers only where deployment qualifies;
- test stable on major desktop/mobile OSes affected by the shared code;
- rerun after component updates and stable releases;
- apply the novelty gate against issue trackers, regression tests, fuzz bug references, and advisories.

## Common Pitfalls

1. Finding a bug only with `--no-sandbox` or unsafe feature flags.
2. Treating renderer RCE as a sandbox escape.
3. Fuzzing generated IPC stubs while missing receiver semantics.
4. Ignoring navigation and object-lifetime identity.
5. Claiming origin impact from process placement alone.
6. Testing Nightly but not current stable.
7. Using an embedded browser version with negligible deployment.
8. Starting exploit work before minimizing the root cause.

## Verification Checklist

- [ ] Latest stable shipped bytes and source revision pinned
- [ ] Normal sandbox and feature configuration preserved
- [ ] Web/renderer-to-sink reachability graph complete
- [ ] Trust owner identified for every validation
- [ ] Renderer, IPC, origin, and OS boundaries classified separately
- [ ] Harness reaches shipped implementation
- [ ] Root cause minimized with negative control
- [ ] Sandbox capability gain or native primitive proven
- [ ] Mitigations and chain requirements assessed
- [ ] Stable-version and novelty gates rerun

