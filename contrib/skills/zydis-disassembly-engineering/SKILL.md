---
name: zydis-disassembly-engineering
description: "Use when decoding/encoding x86/x64 with Zydis in C/C++ or terminal tests."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: re
  upstream: C:\Users\Admin\.agents\skills\zydis-disassembly-engineering\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\zydis-disassembly-engineering\SKILL.md

# Zydis Disassembly Engineering

## Overview
Zydis is a fast, dependency-light x86/x86-64 decoder, formatter, and encoder library. Use it when exact operand metadata, register/flag access, instruction fragment offsets, or a small embeddable decoder matters. Zydis only supports x86-family instructions; route other architectures elsewhere.

## When to Use
Use for C/C++ tooling, scanners, relocators, instrumentation, hook/trampoline validation, binary transformers, and decoder cross-checks. Do not infer control-flow safety from successful decoding alone.

## Local WSL Setup
Observed Ubuntu package version: Zydis 4.1.1 (`libzydis-dev`, `zydis-tools`). The Ubuntu package provides headers, `libZydis.so`, and CMake config under `/usr/lib/x86_64-linux-gnu/cmake/zydis/`, but no `zydis.pc` file. Official stable docs: `https://doc.zydis.re/v4.1.1/html/`; project: `https://github.com/zyantific/zydis`; overview: `https://zydis.re/`.

Always compile against the installed headers and report the package version with `dpkg-query -W libzydis-dev`; do not mix master-branch 5.x examples with the stable 4.1.1 API.

## Decoder Workflow
1. Define machine mode and stack/address width from the actual binary, not the host process.
2. Bound the input buffer to mapped readable bytes.
3. Initialize the decoder and formatter once; check every status code with `ZYAN_SUCCESS`.
4. Decode one instruction at a time and advance by `instruction.length` only after success.
5. Decode operands when semantics matter; do not parse formatted text.
6. Compute absolute addresses using Zydis helpers and the instruction runtime address.
7. Stop or resynchronize explicitly on invalid/truncated input; never loop without progress.

Done when every consumed byte belongs to a validated instruction and failures have deterministic handling.

## Stable v4 Skeleton
Verify exact signatures against installed headers before adapting:
```c
#include <Zydis/Zydis.h>

ZydisDecoder decoder;
ZydisFormatter formatter;
ZydisDecoderInit(&decoder, ZYDIS_MACHINE_MODE_LONG_64,
                 ZYDIS_STACK_WIDTH_64);
ZydisFormatterInit(&formatter, ZYDIS_FORMATTER_STYLE_INTEL);

ZydisDecodedInstruction insn;
ZydisDecodedOperand operands[ZYDIS_MAX_OPERAND_COUNT];
if (ZYAN_SUCCESS(ZydisDecoderDecodeFull(
        &decoder, data, data_len, &insn, operands))) {
    char text[256];
    ZydisFormatterFormatInstruction(
        &formatter, &insn, operands, insn.operand_count_visible,
        text, sizeof(text), runtime_address, ZYAN_NULL);
}
```

Treat this as version-specific code and compile it; do not claim correctness from visual inspection.

## Operand and Address Semantics
- Distinguish `operand_count` from `operand_count_visible`.
- Inspect operand type, size, visibility, actions, register, memory base/index/scale/displacement, immediate signedness/relativity, and encoding offsets.
- For RIP/EIP-relative memory and relative branches, use the absolute-address helper rather than manual arithmetic.
- Track implicit operands and CPU flags when building data-flow or relocation logic.
- Preserve instruction runtime address separately from buffer address.

## Encoder and Relocation Rules
Encoding equivalent text is not proof of semantic equivalence. For approved code generation or relocation:
1. Build an encoder request from explicit semantics.
2. Encode into a bounded buffer and check status/length.
3. Decode the emitted bytes again.
4. Compare mnemonic, operand kinds/widths, relative targets, register/flag effects, and memory address calculation.
5. Test boundary values for displacement/immediate widths.

For trampolines, account for relative control flow, RIP-relative data, short-branch expansion, instruction boundaries, unwind metadata, and atomic patch strategy. Zydis decodes/encodes; it does not make relocation safe automatically.

## Build and Verification
Direct link against the Ubuntu package:
```bash
dpkg-query -W libzydis-dev
cc -std=c11 demo.c -lZydis -o demo
./demo
```

CMake:
```cmake
find_package(Zydis CONFIG REQUIRED)
target_link_libraries(tool PRIVATE Zydis::Zydis)
```

Do not use `pkg-config zydis` on this host: the observed Ubuntu package has no `.pc` file. Cross-check a focused byte sequence with `ZydisInfo`/`ZydisDisasm` if available and at least one independent decoder (`rasm2`, `objdump`, or IDA) for critical work.

## Fuzz and Boundary Cases
Include empty/truncated buffers, invalid encodings, prefixes, maximum 15-byte instructions, mode changes, relative branches, RIP-relative memory, AVX/EVEX/APX-relevant inputs for the linked version, and page-end reads. Assert forward progress and no out-of-bounds access.

## Common Pitfalls
1. Copying Zydis 5.x/master API into a 4.1.1 build.
2. Parsing formatter strings instead of operand structures.
3. Ignoring implicit operands and flag effects.
4. Manually calculating relative targets and getting width/sign extension wrong.
5. Decoding past the readable buffer or across a page boundary.
6. Assuming successful decode means a safe instruction to relocate or execute.
7. Failing to round-trip encoded bytes through the decoder.

## Verification Checklist
- [ ] installed Zydis version and architecture mode recorded
- [ ] source compiled against current headers with no ignored status codes
- [ ] input bounds and decode-failure progress are deterministic
- [ ] semantic logic uses operand metadata, not formatted text
- [ ] relative/absolute addresses computed with helpers
- [ ] encoded output round-tripped and compared semantically
- [ ] critical cases cross-checked with an independent decoder
- [ ] no patch/execution performed without explicit direction

