---
name: virtualization-escape-research
description: "Use when hunting new guest-to-host, container-to-hypervisor, or nested-virtualization vulnerabilities in the latest stable hypervisor, emulator, cloud VM stack, virtual device, paravirtual driver/backend, guest-tools integration, management plane, snapshot/migration path, or hardware-accelerated virtualization interface."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: imported
  upstream: C:\Users\Admin\.agents\skills\virtualization-escape-research\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\virtualization-escape-research\SKILL.md

# Virtualization Escape Research

## Objective

Find latest-stable vulnerabilities where a malicious guest or isolated workload gains host code execution, host memory access, cross-VM access, management-plane control, or a meaningful hypervisor boundary bypass.

Prioritize default virtual hardware and cloud/enterprise configurations over obscure optional devices.

## Phase 1: Pin the Virtualization Stack

Record every independently versioned layer:

- product/hypervisor build and host OS/kernel;
- user-mode device emulator and libraries;
- kernel acceleration module;
- firmware/UEFI and virtual chipset/machine type;
- guest tools, shared-folder/clipboard/graphics agents;
- paravirtual frontend and host backend versions;
- virtual GPU/media/USB/network/storage device models;
- management daemon/API;
- nested virtualization and hardware microcode state;
- VM configuration, device list, and feature flags.

Archive configuration and hashes. “Latest product version” is insufficient if the vulnerable device backend is separately packaged.

Completion criterion: a clean VM can be recreated with the exact virtual hardware and stack versions.

## Phase 2: Enumerate Guest-Controlled Interfaces

Map:

- port I/O and MMIO registers;
- PCI config space, BARs, MSI/MSI-X, capabilities;
- DMA descriptors, rings, queues, scatter/gather lists;
- virtio/vhost, Xen, Hyper-V, VMware, and platform-specific hypercalls;
- emulated USB, audio, display, GPU, network, storage, SCSI/NVMe, and legacy devices;
- shared memory, ballooning, filesystem, clipboard, drag/drop, and guest-agent channels;
- VM exits caused by MSR, CPUID, APIC, instruction emulation, page faults;
- snapshot/save/restore, live migration, device hotplug, suspend/resume;
- nested VMCS/VMCB and enlightened interfaces;
- management APIs reachable from guest agents or virtual networks.

For each interface record host handler, process/privilege, guest-controlled fields, memory model, and default presence.

Completion criterion: all default guest-to-host interfaces map to handler code and trust level.

## Phase 3: Model Memory Ownership

For each queue or descriptor chain define:

- guest physical address translation and pinning;
- length/count arithmetic and maximums;
- descriptor ownership transitions;
- indirect descriptors and recursion limits;
- mapping lifetime across async host I/O;
- IOMMU and bounce-buffer behavior;
- host pointers/cookies stored in guest-visible memory;
- reset, cancellation, hot-unplug, and migration semantics;
- concurrency between vCPU, I/O, and worker threads.

Write an ownership timeline. Most useful bugs violate it during reset, async completion, or state restoration.

## Phase 4: Harness and Snapshot Fuzzing

Choose among:

- direct unit harness for device read/write handlers;
- guest kernel/user driver generating MMIO/PIO/descriptor traffic;
- hypercall grammar fuzzer;
- QEMU/libFuzzer-style device harness;
- VM snapshot loop restoring immediately before the handler;
- migration/saved-state mutator;
- differential execution across machine types or versions.

Feedback:

- host sanitizers and crash dumps;
- device-model coverage;
- VM-exit/handler coverage;
- host process syscalls and allocations;
- guest-observed completion/status anomalies;
- invariant assertions in instrumented builds.

Keep the guest generator deterministic and log every operation needed to replay after host crash.

Completion criterion: a minimized operation sequence reproduces from a clean snapshot.

## Phase 5: High-Yield Campaigns

### Descriptor and DMA

- cyclic/overlapping chains;
- zero, huge, and wrapping lengths;
- indirect table nesting;
- descriptor mutation after validation;
- guest page unmap/remap during async I/O;
- inconsistent queue size/index/event values;
- partial completion and reset races.

### Device lifecycle

- reset during pending DMA;
- hot-unplug while callbacks are queued;
- suspend/resume with stale host objects;
- error recovery and timeout paths;
- interrupt delivery after teardown;
- frontend/backend reconnect and negotiation downgrade.

### State serialization

- malformed snapshot/migration sections;
- version skew and optional field mismatch;
- integer truncation across host architectures;
- restored pointers, indexes, and lengths not revalidated;
- destination host capabilities differing from source.

### Instruction and nested virtualization

- decode length, prefixes, segment/address modes;
- invalid VMCS/VMCB combinations;
- nested intercept merging;
- synthetic MSR and hypercall buffers;
- state transitions during exception/NMI injection.

### Integration channels

- shared folders and path translation;
- clipboard/drag/drop object lifetimes;
- graphics command buffers and shader/media parsers;
- guest agent message deserialization and update paths.

Completion criterion: campaigns are separated so one noisy device does not starve others.

## Phase 6: Escape Triage

Determine:

- crash context and root cause in host code;
- host process sandbox, user, seccomp/token/profile, and namespace;
- controlled read/write/free, host pointer leak, or logic capability;
- guest-to-host address predictability and mitigations;
- whether an emulator-process compromise still requires a second sandbox escape;
- cross-VM or management socket reachability;
- default device presence and cloud applicability;
- behavior with IOMMU, confidential-computing, and hardened configurations.

Do not call device-emulator code execution a full host escape if the emulator is intentionally sandboxed; report the remaining boundary.

## Phase 7: Variants

Search:

- sibling devices using the same descriptor helper;
- userspace and kernel backends;
- legacy and modern machine types;
- host architecture ports;
- migration load paths corresponding to live MMIO handlers;
- cloud forks and downstream backports;
- nested versus non-nested code;
- guest tools sharing serializers with management services;
- fixes applied to one queue direction or device mode only.

## Common Pitfalls

1. Fuzzing optional devices absent from widespread deployments.
2. Recording only top-level hypervisor version.
3. Losing the final guest operation sequence after a host crash.
4. Ignoring emulator sandboxing when stating impact.
5. Mutating descriptors without modeling ownership timing.
6. Testing instrumented debug code but not latest stable shipped bytes.
7. Treating guest kernel compromise as a prerequisite without stating it.
8. Ignoring snapshot/migration and reset paths.

## Verification Checklist

- [ ] Complete stack and virtual hardware versions recorded
- [ ] Default guest-host interface map complete
- [ ] Descriptor/memory ownership timelines defined
- [ ] Snapshot or harness replay is deterministic
- [ ] Host sanitizer/crash evidence captured
- [ ] Root cause and controlled primitive established
- [ ] Emulator sandbox and remaining host boundary assessed
- [ ] Default deployment/cloud relevance documented
- [ ] Lifecycle and migration variants tested
- [ ] Latest-stable and novelty gates pass

