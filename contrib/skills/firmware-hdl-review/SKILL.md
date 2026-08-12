---
name: firmware-hdl-review
description: "Use when reviewing firmware/HDL changes across registers, SystemVerilog, DMA, sim, CI."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: re
  upstream: C:\Users\Admin\.agents\skills\firmware-hdl-review\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\firmware-hdl-review\SKILL.md

# Firmware and HDL Code Review

Use this skill when a review crosses hardware register models, HDL generators/templates, generated RTL, device emulators, and driver-facing tests.

## Review goals

Find merge-blocking issues in:

- Register offsets and access semantics
- Descriptor ownership and ring state
- DMA bounds, errors, byte enables, and address width
- RTL handshake and FSM behavior
- Interrupt cause/mask/ack/MSI/MSI-X paths
- Cross-model consistency
- Donor acquisition safety: live-device side effects, opt-in destructive operations, and restoration
- Untrusted artifact safety: schema/topology validation, checked arithmetic, allocation and path bounds
- Simulation, synthesis, lint, CI, and packaging integration

## Workflow

### 1. Pin the baseline

1. Resolve and record the exact base SHA.
2. Inspect worktree/index state separately.
3. Compare candidate files against clean base objects such as `git show <base>:<path>`, not against a possibly dirty checkout.
4. Normalize CRLF/LF before producing the delta inventory.
5. Do not modify source during an audit unless explicitly asked.

### 2. Inventory every layer

Map each changed behavior through:

1. Authoritative hardware or upstream-driver definitions
2. Device profile and BAR/register model
3. Generator code and HDL templates
4. Generated HDL output
5. Software/VFIO behavior model
6. Unit tests, HDL lint, simulation, synthesis, and CI targets

Tests that copy constants from the implementation are not independent evidence. Validate offsets and semantics against an authoritative source.

### 3. Review contracts, not files in isolation

For each register or descriptor field, write down:

- Address and width
- Reset, RW, W1C, read-clear, and FSM ownership
- Which side owns and advances head/tail pointers
- Legal ring size, alignment, and wrap behavior
- Address width and upper-half programming
- Completion/error semantics
- Interrupt cause and acknowledgement path

Then confirm every implementation layer follows the same contract.

### 4. Exercise generated RTL

- Lint the new module alone for fast syntax feedback.
- Generate a real fixture and lint the complete top-level design.
- Confirm source allowlists/manifests include every new generated module.
- Run a functional simulation for descriptor, DMA, and interrupt behavior.
- Inspect synthesis-sensitive issues: multiple drivers, inferred latches, width truncation, unreachable states, undeclared/black-box modules, and unsupported constructs.

### 5. Audit DMA safety

Check zero and malformed addresses, alignment, range, upper 32-bit address handling, maximum lengths, destination capacity, counter widths, final byte enables, timeout/error propagation, and partial-completion behavior. A device-emulation feature must not silently turn malformed descriptors into writes through stale or zero addresses.

### 6. Differentially test models

Use a common golden byte sequence and transaction trace for RTL and software emulation. Compare exact payload bytes, descriptor status, heads/tails, interrupt causes, and error behavior. Include wrap, simultaneous submissions, busy-time submissions, nonzero upper addresses, alternate BARs/BIRs, partial DWORDs, and injected DMA failures.

### 7. Audit donor acquisition and untrusted inputs

- Trace every live-device write, including driver bind/unbind, FLR, PMCSR/runtime-PM changes, PCI Command writes, BAR probes, doorbells, and read/write ioctls. Diagnostic and default collection paths should be read-only; destructive actions require explicit, narrowly scoped opt-in.
- Treat restore-after-write as insufficient for W1C, read-clear, doorbell, reset, and DMA-trigger registers: their side effects are not reversible by writing the original value back.
- Validate imported donor JSON before generation: strict schema/EOF, bounded file and decoded payload sizes, config-space size/word count, BAR/BIR topology, 64-bit BAR pairing, profile offsets, MSI-X ranges, and consistency with authoritative config bytes.
- Keep size arithmetic unsigned and checked until after capping. Compatibility overrides such as `--force` must never bypass overflow, topology, allocation, or path-safety limits.
- Bound reset images and generated buffers to the captured/modelled window rather than the full advertised BAR aperture.
- Before recursive cleanup, prove the output directory is tool-owned; reject roots, ambiguous existing directories, and unsafe symlink resolution.
- See `references/donor-acquisition-input-safety.md` for concrete review patterns and regression tests.

### 8. Check integration and merge risk

- Identify what the root test target actually runs; do not assume separate cocotb/C/HDL targets are included.
- Check generated-file validators, lint source patterns, feature flags, release manifests, and archive required-file lists.
- Test textual applicability against the clean base/index, not a dirty worktree.
- Report literal merge conflicts separately from behavioral regressions.

## Output format

Lead with a verdict: approve, comment, or block merge. Prioritize findings as Critical, High, Medium, and Low. Every finding must include:

- Exact candidate file and line range
- Concrete failure mode and affected environment
- Evidence or reproduction result
- Root-cause fix, not only a symptom workaround
- Missing regression test

End with commands actually run, suites not run and why, baseline SHA, conflict status, and whether any files were modified.

## Reference

See `references/firmware-hdl-checklist.md` for detailed pitfalls and boundary cases.

