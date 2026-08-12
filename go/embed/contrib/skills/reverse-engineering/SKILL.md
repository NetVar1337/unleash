---
name: reverse-engineering
description: "Reverse engineering workflow for binaries, malware, firmware, protocols, and anti-cheat systems. Covers static analysis, dynamic analysis, decompilation, binary instrumentation, and protocol RE. Invoke with /reverse-engineering or when the task involves RE work."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: re
  upstream: local:C:\Users\Admin\.claude\skills
---

> Bundled with Unleash skills pack. Upstream: local:C:\Users\Admin\.claude\skills

# Reverse engineering workflow

## Activation

Use when the task involves reverse engineering binaries, analyzing malware,
RE firmware, reversing protocols, anti-cheat analysis, or binary
instrumentation.

## Workflow by target type

### Windows PE binary

1. **Triage:** `file`, `strings`, `sigcheck`, `pestudio`, `Detect It Easy`.
2. **Static analysis:**
   - IDA Pro / Ghidra / Binary Ninja for disassembly + decompilation.
   - Map imports → identify API usage patterns.
   - Identify packer/protector (UPX, Themida, VMProtect, Enigma).
   - Unpack if needed (x64dbg + Scylla, manual OEP finding).
3. **Dynamic analysis:**
   - x64dbg / WinDbg for user-mode.
   - API Monitor / Procmon for behavioral overview.
   - ETW tracing for kernel interactions.
4. **Annotate:** Name functions, structs, globals. Apply FLIRT signatures.
   Create type libraries for known SDK structs.

### Linux ELF binary

1. **Triage:** `file`, `readelf -a`, `strings`, `ldd`, `checksec`.
2. **Static:** Ghidra / IDA / radare2. Map PLT/GOT, identify libc calls.
3. **Dynamic:** GDB + GEF/pwndbg, `ltrace`, `strace`, `perf`.
4. **Instrumentation:** Frida, DynamoRIO, Intel Pin for tracing/hooking.

### Malware analysis

1. **Sandbox first:** Run in isolated VM (FlareVM / REMnux), capture
   PCAP + screenshots + memory dump.
2. **Static triage:**
   - Hash (MD5/SHA256), VirusTotal, MalwareBazaar.
   - Strings, imports, sections, resources.
   - Identify family via YARA rules or known patterns.
3. **Behavioral analysis:**
   - File system: dropped files, persistence (Run keys, services,
     scheduled tasks, WMI subscriptions).
   - Network: C2 protocol, domains, IPs, JA3/JA4 fingerprint.
   - Registry: modifications, config storage.
   - Process: injection targets, hollowing, doppelgänging.
4. **Deep analysis:**
   - Unpack/decrypt payloads (identify crypto: XOR, RC4, AES, ChaCha20).
   - Reverse C2 protocol for detection signatures.
   - Map kill chain: initial access → persistence → C2 → objective.
5. **Deliverables:** IOC list, YARA rules, network signatures, technical
   writeup.

### Firmware analysis

1. **Extraction:** `binwalk`, `firmware-mod-kit`, `uefi-firmware-parser`,
   chip-off / SPI dump.
2. **Filesystem:** SquashFS, JFFS2, CramFS, UBIFS → `unsquashfs`,
   `jefferson`.
3. **Analysis:**
   - Identify init scripts, services, binaries.
   - Reverse proprietary protocols (UART, SPI, I2C, JTAG).
   - Check for hardcoded credentials, backdoors, debug interfaces.
   - Analyze bootloader (U-Boot, UEFI) for secure boot bypass.
4. **Emulation:** QEMU system emulation, `firmadyne`, `FAT` (Firmware
   Analysis Toolkit).

### Protocol reverse engineering

1. **Capture:** Wireshark / tcpdump / mitmproxy / Burp Suite.
2. **Identify framing:** Message boundaries, length fields, delimiters.
3. **Map fields:** Magic bytes, version, type, length, checksum, payload.
4. **Identify encoding:** Plain, XOR, zlib, protobuf, MessagePack, custom.
5. **Identify crypto:** TLS, DTLS, custom handshake. Extract keys from
   memory if needed (SSLKEYLOGFILE, Frida hook on `SSL_write`).
6. **Replay/fuzz:** Scapy, Boofuzz, custom fuzzer for stateful protocols.
7. **Document:** Field table, state machine diagram, sample captures.

### Anti-cheat / game security

1. **Identify protection:** Kernel driver (EAC, BE, Vanguard, Ricochet),
   user-mode DLL, hypervisor-based.
2. **Driver analysis:**
   - Load in WinDbg, map dispatch routines.
   - Identify callback registrations (process, thread, image, registry).
   - Find scanning logic (memory scans, handle scans, module checks).
   - Locate integrity checks (code checksums, import table validation).
3. **Bypass strategy:**
   - Disable callbacks (see kernel-dev skill).
   - Spoof scanned regions (VAD manipulation, EPT hooks).
   - Hide process/module from enumeration.
   - Patch integrity checks or hook scan functions.
4. **User-mode analysis:**
   - Identify overlay / rendering hook points.
   - Map shared memory regions for data exchange.
   - Reverse packet format for network-based games.

## Tooling reference

| Category | Tools |
|---|---|
| Disassembly / decompilation | IDA Pro, Ghidra, Binary Ninja, radare2 |
| User-mode debugging | x64dbg, WinDbg, GDB + GEF |
| Kernel debugging | WinDbg (kd/net), QEMU + GDB |
| Binary instrumentation | Frida, Intel Pin, DynamoRIO, TinyInst |
| Network capture | Wireshark, tcpdump, mitmproxy, Burp Suite |
| Fuzzing | AFL++, libFuzzer, Boofuzz, kAFL, syzkaller |
| Malware sandbox | FlareVM, REMnux, Cuckoo, CAPE |
| Firmware | binwalk, Ghidra, QEMU, firmadyne |
| PE analysis | pestudio, PE-bear, CFF Explorer, sigcheck |
| ELF analysis | readelf, objdump, checksec, patchelf |
| YARA / detection | YARA, ClamAV, Suricata, Zeek |

## Verification checklist

- [ ] All major functions identified and named
- [ ] Data structures reconstructed with correct field offsets
- [ ] Control flow mapped (entry points, dispatch, state machines)
- [ ] Crypto / encoding identified and parameters extracted
- [ ] Network protocol documented with field table (if applicable)
- [ ] IOCs / signatures produced (if malware)
- [ ] Findings reproducible from clean state
