---
name: vuln-research
description: "0-day vulnerability discovery workflow. Covers fuzzing strategies, variant analysis, static analysis for bug classes, attack surface mapping, PoC development, and responsible triage. Invoke with /vuln-research or when the task involves finding new vulnerabilities."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: security
  upstream: C:\Users\Admin\.claude\skills\vuln-research\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.claude\skills\vuln-research\SKILL.md

# 0-day vulnerability discovery workflow

## Activation

Use when the task involves discovering new vulnerabilities, fuzzing, variant
analysis, attack surface mapping, or 0-day research.

## Research phases

### 1. Target selection & attack surface mapping

**Identify attack surface:**
- File parsers (image, font, document, media, archive formats)
- Network protocols (TLS, HTTP/2, QUIC, DNS, SMB, RDP, custom)
- Kernel interfaces (syscalls, IOCTLs, netlink, eBPF, /proc, /sys)
- IPC (D-Bus, COM, RPC, shared memory, named pipes)
- Browser engines (V8, SpiderMonkey, JavaScriptCore, Blink, Gecko)
- Game engines / anti-cheat drivers (IOCTL dispatch, shared memory,
  network protocol)
- Firmware (UEFI, BMC, embedded RTOS, bootloaders)
- Virtualization (hypercall interfaces, virtio, device emulation)

**Map the surface:**
1. Enumerate entry points: syscalls, IOCTLs, exported functions, protocol
   handlers, file format parsers.
2. Trace data flow from untrusted input to sensitive operations.
3. Identify trust boundaries: user→kernel, guest→host, network→parser,
   file→engine.
4. Prioritize by: complexity of input, history of bugs in similar code,
   privilege gained, reachability.

### 2. Static analysis for bug classes

**Memory corruption:**
- Buffer overflow: fixed-size stack/heap buffers with unchecked lengths.
  Look for `memcpy`, `strcpy`, `sprintf`, `ReadFile` into fixed buffers.
- Use-after-free: object freed but pointer retained; callback after
  destruction; async completion after cancel.
- Double-free: error paths that free then fall through to normal cleanup.
- Integer overflow: size calculations (`width * height * bpp`), length
  fields parsed from untrusted input, `malloc(n * elem_size)`.
- Type confusion: cast without validation, union misuse, virtual call on
  wrong type, container_of with wrong offset.
- Uninitialized memory: struct allocated without zeroing, conditional
  initialization paths.

**Logic bugs:**
- TOCTOU (time-of-check-time-of-use): validate then use with gap between.
- Race conditions: concurrent access without locks, signal handlers
  modifying shared state.
- Reference counting: missing increment/decrement, overflow of refcount.
- State machine violations: operation allowed in wrong state, missing
  state transition.
- Permission bypass: missing access check, check on wrong object,
  impersonation.

**Variant analysis process:**
1. Take a known CVE in the target or similar codebase.
2. Identify the root cause pattern (not the specific instance).
3. Search for the same pattern across the codebase:
   - Same API misuse in other callers.
   - Same missing check in sibling functions.
   - Same logic in different platform/version.
4. Tools: CodeQL, Semgrep, grep + coccinelle, IDA FLIRT + cross-refs.

### 3. Fuzzing

**Strategy selection:**

| Target type | Fuzzer | Approach |
|---|---|---|
| File parser | AFL++, libFuzzer, Honggfuzz | Corpus of valid files, mutation-based |
| Network protocol | AFL++ with QEMU, Boofuzz, Scapy | Stateful, grammar-based |
| Kernel syscall | syzkaller, Trinity | Syscall description files |
| Kernel IOCTL | kAFL, custom harness | IOCTL code + buffer mutation |
| Browser engine | Fuzzilli, Domato, Grizzly | JS/DOM grammar |
| Library API | libFuzzer, AFL++ persistent | In-process harness |
| Game protocol | Custom, Boofuzz | Replay + mutate captured packets |

**Harness writing:**
```c
// libFuzzer harness for a parser
int LLVMFuzzerTestOneInput(const uint8_t *data, size_t size) {
    // Initialize parser state (once)
    static bool initialized = init_parser();

    // Feed fuzzed data
    parse_input(data, size);

    // Cleanup per-iteration state
    reset_parser_state();
    return 0;
}
```

**Kernel fuzzing (syzkaller):**
1. Write syscall descriptions (`.txt`):
   ```
   ioctl$MY_DRIVER(fd fd_mydev, cmd int32, arg ptr[in, my_struct])
   my_struct {
       field1  int64
       field2  ptr[out, buffer]
       size    len[field2]
   }
   ```
2. Build kernel with KASAN, KMSAN, UBSAN, lockdep.
3. Run syz-manager with QEMU or bare metal.
4. Triage crashes: KASAN report → root cause → exploitability.

**Coverage-guided tips:**
- Start with a seed corpus of valid inputs (not random).
- Use sanitizers (ASan, MSan, UBSan, KASAN) to detect subtle bugs.
- Run multiple parallel instances with different seeds.
- Minimize corpus periodically (`afl-cmin`).
- For structure-aware fuzzing: custom mutator or grammar (libprotobuf-mutator).

### 4. Dynamic analysis & instrumentation

- **Code coverage:** Intel PT, DynamoRIO (drcov), kcov (kernel).
- **Taint tracking:** Intel Pin + libdft, DynamoRIO + drmemtrace.
- **API hooking:** Frida (user), kprobes/ftrace (kernel), Detours (Windows).
- **Differential testing:** Run same input on two versions/implementations,
  diff behavior.
- **Snapshot debugging:** QEMU snapshot + replay for deterministic debugging
  of race conditions.

### 5. PoC development

1. **Minimize:** Reduce triggering input to smallest case
   (`afl-tmin`, `casr`, manual).
2. **Stabilize:** Ensure reliable trigger (control heap layout, timing).
3. **Classify:** Determine bug class and severity.
4. **Assess exploitability:**
   - Can you control the corruption (offset, size, content)?
   - What objects are adjacent / reclaimable?
   - Are mitigations present (ASLR, CFI, SMEP, CET)?
5. **Document:** Root cause, trigger path, affected versions, impact,
   minimal PoC.

### 6. Triage & reporting

- Severity: CVSS score based on attack vector, complexity, privileges,
  impact.
- Affected versions: bisect to find introduction commit/release.
- Report: root cause analysis, PoC, suggested fix, timeline.

## Tooling reference

| Category | Tools |
|---|---|
| Coverage fuzzing | AFL++, libFuzzer, Honggfuzz, kAFL |
| Generation fuzzing | Boofuzz, Scapy, Fuzzilli, Domato |
| Kernel fuzzing | syzkaller, Trinity, custom IOCTL fuzzer |
| Static analysis | CodeQL, Semgrep, Coccinelle, IDA/Ghidra |
| Sanitizers | ASan, MSan, UBSan, KASAN, KMSAN, lockdep |
| Instrumentation | Intel Pin, DynamoRIO, Frida, kprobes |
| Triage | CASR, exploitable (GDB), crashwalk |
| Minimization | afl-tmin, casr-afl, libFuzzer -minimize_crash |

## Verification checklist

- [ ] Attack surface fully enumerated and prioritized
- [ ] Bug class patterns searched across codebase (variant analysis)
- [ ] Fuzzer harness covers target entry point with sanitizers enabled
- [ ] Seed corpus is valid, diverse, and minimized
- [ ] Crashes triaged: unique root causes identified
- [ ] PoC minimized and reproducible
- [ ] Exploitability assessed (controlled corruption, adjacent objects)
- [ ] Affected version range determined

