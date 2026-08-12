---
name: hardware-firmware-validation
description: "Use for FPGA/PCIe/DMA/HDL validation across simulators, manifests, and CI."
version: 1.2.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: re
  upstream: C:\Users\Admin\.agents\skills\hardware-firmware-validation\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\hardware-firmware-validation\SKILL.md

# Hardware and Generated-Firmware Validation

Use this skill when a change crosses software and hardware-model boundaries: generators, emitted HDL, host-side reference models, descriptors/registers, simulation harnesses, manifests, Tcl/project source lists, or CI.

## Core principle

A generated HDL feature is not complete when the template renders. It is complete only when:

1. The software model and HDL model share the same register/protocol contract.
2. Every generated module is emitted, declared in manifests, included in project/lint source closure, and instantiated conditionally.
3. The HDL compiles in the same full design context used by CI.
4. Behavioral simulation proves state transitions and host-memory effects.
5. Existing classes/boards still pass their regression matrix.

## 1. Establish a trustworthy baseline

Before importing changes:

1. Record the upstream commit and branch.
2. Check repository instructions, CI jobs, manifests, source-discovery scripts, and existing known-limitations documents.
3. Run the strongest available baseline tests before editing.
4. Use a feature branch; never develop directly on the upstream default branch.
5. If the working checkout is on Windows but validation is Linux-centric, validate in a clean Linux-native clone or worktree. Do not diagnose CRLF artifacts as HDL logic failures. If a clean clone is impractical, create a temporary LF-normalized copy of the shell entrypoint at the same relative directory depth, execute that copy, and remove it afterward; preserving the relative depth matters when the script derives the repository root from `BASH_SOURCE`. Do not rewrite tracked files merely to make one validation command executable.
6. In a shared dirty worktree, record hashes or modification times for files under review, re-read them after long-running tests, and base line-level findings on the final observed snapshot. Concurrent edits can otherwise invalidate an apparently precise audit.

When comparing an exported early-access directory with Git:

- Exclude `.git`, submodule contents, build directories, simulator output, generated fixture files, and test-result XML.
- Normalize CRLF/LF before hashing or diffing text.
- Compare semantic file contents first; an archive extracted on Windows can make every text file appear changed.
- Determine whether the export is upstream plus a small delta before copying files wholesale.

See `references/pcie-generated-firmware.md` for a detailed cross-layer checklist. For repository-wide technical gap analyses based on known issues, TODOs, issue/PR evidence, code, tests, CI omissions, and release gates, use `references/validation-gap-analysis.md`; it includes the required evidence hierarchy, proposal format, and hardware-free acceptance strategy. For xHCI/NVMe audits, use `references/xhci-nvme-hardening.md` for exact programming-interface gating, queue/CQ ownership, PRP safety, FIFO accounting, MSI-X, reset coherence, and failure-injection tests. For Intel Ethernet descriptor engines, use `references/e1000e-dma-integration.md` for the canonical register map, ring ownership, interrupt semantics, and boundary-test matrix. For broad donor support, emulation tiers, final-configuration downgrades, and machine-readable support reports, use `references/donor-family-capability-model.md`. When the user explicitly requests PR visibility before all gates pass, use `references/draft-pr-with-blockers.md` for the fail-transparent draft workflow and cross-fork read-back checks.

## 2. Map the cross-layer contract

Write down the contract before reviewing implementation:

| Layer | Required agreement |
|---|---|
| Donor/device selection | Vendor, device, class, revision, BAR/BIR, capability mode |
| Host model | Register offsets, reset values, ownership, descriptor semantics |
| Generator config | Exact feature predicate and supported device family |
| HDL endpoint | Ports, FSM ownership, byte enables, reset/quiesce behavior |
| DMA bridge | Request lengths, tags, completions, timeouts, TLP formatting |
| Interrupt path | MSI/MSI-X enable, mask, pending/PBA, delivery acknowledgement |
| Artifacts | Output list, manifest, validator, Tcl/project source inclusion |
| Tests | Unit generation, lint, behavioral simulation, negative cases |

### Never gate protocol behavior by class alone

A PCI class code does not define a register layout. Ethernet, storage, USB, audio, and GPU devices from different vendors can have incompatible protocols despite sharing a class.

Gate class-specific engines using a tested vendor/device-family predicate. For unsupported families:

- Keep donor-backed/static behavior, or
- Fail generation with a precise unsupported-model message.

Do not silently apply an Intel-style descriptor engine to Realtek, Broadcom, or unrelated devices merely because all are Ethernet controllers.

### Separate family detection from activated capability

A recognized family may safely receive its register model while a deeper engine remains disabled by the final build configuration. Determine the advertised support level only after BAR/BIR selection, aperture/resource checks, interrupt-mode selection, stock/static mode, and all generator predicates have run.

Emit a manifest-covered support report for every build with family, level (`identity`, `registers`, `behavior`, or `dma`), validation state, and explicit limitations. Downgrade the report when the specialized engine was requested but not actually generated. A validated level with known limitations is not “complete.” See `references/donor-family-capability-model.md`.

When changing a family predicate, audit every conditional block in the template—not only module instantiation. A leftover class-wide reset/FSM block can reference registers absent from the newly selected family model and will often appear only during full-design elaboration.

## 3. Review DMA and descriptor semantics

For every ring:

- Identify software-owned and device-owned pointers.
- Define what head and tail mean, including the `head == tail` case.
- Validate ring length, alignment, entry count, head/tail bounds, and base address.
- Process the descriptor at the device-owned head, not at the software tail.
- Advance only device-owned pointers.
- Handle wraparound explicitly.
- Reject null buffer addresses and implausible lengths before issuing DMA.
- Bound writes to the modeled receive-buffer capacity.
- Use `(length + word_bytes - 1) / word_bytes` for rounded-up word counts.
- Apply precise byte enables on partial final words and descriptor subfields.
- Preserve descriptor fields not owned by the device.
- Add timeout/abort recovery so FSMs cannot wait forever.

Synthetic traffic and TX loopback can race for the same RX descriptor. Prefer a pending event with a bounded delay and give an immediately submitted TX packet priority, rather than consuming every newly posted RX descriptor synchronously.

## 4. Verify packet constants as bytes

Do not review packet ROM constants by eye.

Reconstruct generated DWORDs into bytes using the bus/host endianness, then verify:

- Destination/source MAC boundaries
- EtherType
- ARP hardware/protocol types, address lengths, and opcode
- IPv4 version, IHL, total length, protocol, source/destination
- IPv4 header checksum
- ICMP type/code/checksum
- Descriptor-reported length versus bytes actually written
- Padding versus protocol length

A packet can look plausible as hexadecimal words while fields are shifted across DWORD boundaries.

## 5. Prove generated-source closure

Whenever a new generated module is added, update and test all of these:

- Template embedding/registry
- Generator function
- Conditional generation predicate
- Output-file inventory
- Manifest checksums
- Output validator requirements
- Tcl/Vivado source list
- Verilator/HDL-lint source whitelist or discovery
- Simulation Makefile/source collection
- Early-access/release archive required-file list

A unit test that checks `strings.Contains(generated, "module_name")` is insufficient. Run the full generated design through elaboration so missing modules and unresolved ports are caught.

### Change generated interfaces atomically

Before renaming or adding an HDL port, enumerate the complete producer/consumer chain: module declaration, every instantiation, intermediate wire declaration, template condition, standalone testbench, and full-design wrapper. Apply that interface migration as one coherent change and immediately render plus lint the smallest complete design. Do not leave a working tree with a half-migrated port contract while moving to unrelated fixes; partial template interfaces make subsequent failures noisy and recovery error-prone.

For shared templates, account for every generated specialization. An added output may be safely omitted from a named-port instantiation only when the HDL language and lint policy permit it; otherwise connect or intentionally tie it off in all consumers. Add a negative predicate test so unsupported device configurations do not reference absent ports or modules.

### Avoid generator-delimiter collisions

When HDL is embedded in Go templates, literal SystemVerilog replication syntax can collide with Go's `{{` action delimiter. For example, `{{11{1'b0}}, bit}` may fail during template parsing rather than HDL compilation. Rewrite it as an equivalent form without literal `{{` (for example `{11'b0, bit}`), or emit the construct through a tested template helper. Include template rendering before standalone HDL lint so these failures are classified at the right layer.

## 6. Validation ladder

Run in this order and fix each layer before proceeding:

1. **Formatting and static checks**
   - Language formatter, vet/lint, shell validation, diff checks.
2. **Generator/unit tests**
   - Render templates, assert feature predicates, test unsupported-device exclusion.
3. **Standalone HDL lint**
   - Lint each new module.
4. **Full generated-design lint**
   - Generate every fixture/board matrix cell and elaborate the full controller.
5. **Behavioral simulation**
   - Exercise real host-memory reads/writes, descriptor completion, interrupts, malformed descriptors, reset, and timeout.
6. **Host/reference-model tests**
   - C/C++ sanitizers, valgrind, fuzz smoke, and model parity where applicable.
7. **Full regression suite**
   - Race tests and all existing device classes.
8. **Manifest/archive validation**
   - Verify checksums and required files from a clean output/archive.
9. **Independent review**
   - Review the final diff, not the initial import.
10. **Remote CI**
   - Push to a fork, open the upstream PR, and wait for every required check.

### Do not trust process exit alone

Some simulator wrappers can print a failed test while the outer `make` command exits successfully. Verify the simulator/JUnit result explicitly:

- Confirm expected test count.
- Confirm zero failures and zero errors.
- Confirm the named testcase ran rather than being skipped or filtered out.

Use `references/simulator-result-gating.md` for a fail-closed XML gate, a focused PR-CI pattern, and shared-worktree drift handling.

## 7. Interrupt checklist

“Interrupt handling works” must cover the advertised capability mode:

### MSI

- Event is latched if masked/disabled.
- Enabling MSI with a pending cause eventually delivers a pulse.
- Delivery is not repeated after acknowledgement unless the cause remains pending.

### MSI-X

- Vector selection, table address/data, vector mask, function mask, and PBA are connected.
- Masked events set pending state.
- Unmasking delivers the pending event.
- Delivery completion clears pending state at the correct time.

If only MSI is implemented, gate the feature to MSI-only devices or clearly reject MSI-X configurations. Never silently drop interrupts for donors advertising MSI-X.

## 8. Merge-readiness report

Before claiming ready to merge, report concrete evidence:

- Base and head commits
- Changed files and supported device predicates
- Exact tests run and pass/fail counts
- HDL matrix totals (pass/skip/fail)
- Simulation testcases exercised
- Sanitizer/fuzz/static-analysis results
- Remaining hardware-only validation boundaries
- Fork branch and upstream PR URL
- Remote CI state and mergeability

Do not mark a PR ready for review or merge while known HDL elaboration or behavioral-simulation failures remain. By default, keep fixing locally before publishing.

If the user explicitly asks to open a PR despite documented blockers, use a **draft PR exception**:

1. Run every available independent gate and record exact pass/fail counts.
2. Split the work into reviewable commits, but do not label them verified.
3. Open only a draft; never enable auto-merge.
4. Put failing simulations and unresolved audit findings in a prominent `Known blockers` section.
5. Verify the pushed SHA, upstream base, fork owner/head, draft state, and initial CI checks after creation.
6. Continue treating the branch as non-merge-ready until the blockers are fixed.

A draft is a review/discussion artifact, not a quality bypass.

## Common pitfalls

- Treating line-ending churn as a real 300-file change
- Copying bundled submodule contents from a release archive into the parent repo
- Adding an HDL template without adding it to lint/Tcl source discovery
- Testing only string presence instead of elaboration and simulation
- Mixing register maps from different vendors in one class-wide model
- Using the tail pointer as the current receive descriptor
- Writing complete descriptor DWORDs when only one status byte is device-owned
- Reporting a simulator command as passed without checking its result XML
- Claiming MSI-X support when only legacy MSI is wired
- Declaring merge readiness before rerunning tests after the final fix

