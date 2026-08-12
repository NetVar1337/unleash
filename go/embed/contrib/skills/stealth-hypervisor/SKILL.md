---
name: stealth-hypervisor
description: "Stealth hypervisor development: EPT hooks, SLAT subversion, timing anti-detect, CPUID/MSR hiding, nested virt, anti-AC considerations."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: stealth
  triggers:
    - "EPT stealth"
    - "hypervisor stealth"
    - "CPUID hide"
    - "SLAT hook"
    - "anti-detect HV"
---

# Stealth hypervisor development

Extends `hypervisor-dev` with **detectability** as a first-class constraint.

## Architecture choices
- Type-2 (driver-based) vs Type-1 tradeoffs for gaming/AC hosts
- Intel VT-x + EPT vs AMD-V + NPT
- Single-purpose thin HV (hooks only) vs general virtualization

## Stealth surface
| Signal | Mitigation themes |
|---|---|
| CPUID leaves | Mask hypervisor bit; consistent leaves |
| Timing (RDTSC/rdtscp) | TSC offsetting; careful VMEXIT cost |
| MSR diffs | Handle IA32_FEATURE_CONTROL, DEBUGCTL, LSTAR carefully |
| CR0/CR4/EFER shadows | Keep guest-visible state coherent |
| EPT violations | Coalesce hooks; execute-only pages; split-view |
| Perf counters | Avoid obvious PMU anomalies |
| Nested virt | Detect/handle if guest enables HV |
| DMA | IOMMU awareness when relevant |

## EPT hook craft
- Execute-only + shadowed data pages for stealth reads
- Avoid dual-mapping mistakes that break coherency
- Cache translation invalidations correctly (INVEPT/INVVPID)

## Build/test loop
1. Boot under HV with serial/log channel
2. Self-tests: intercept CPUID, EPT hook NOP sled, hide leaf
3. Run detection suite (public + custom) before AC-specific work
4. Measure VMEXIT rate under game load

## Pair with
`hypervisor-dev`, `game-hacking`, `kernel-dev`, `windows-internals`.
