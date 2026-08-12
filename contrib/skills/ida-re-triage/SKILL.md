---
name: ida-re-triage
description: "Use when triaging an authorized binary/driver/firmware/dump/IDB for RE or vuln research."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: re
  upstream: C:\Users\Admin\.agents\skills\ida-re-triage\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\ida-re-triage\SKILL.md

# IDA RE Triage

Start static, preserve evidence, and make analysis changes only when requested.

1. Confirm target, architecture, scope, and a working-copy location. Record SHA-256, file size, architecture, load base, and tool version.
2. If no IDB is active, open the working copy in IDA and wait for auto-analysis before using `idassistmcp`. Use the configured `idassistmcp` only after it responds.
3. Read first: binary metadata, segments, imports, exports, entry points, strings, functions, code, and xrefs. State addresses and evidence in findings.
4. For Windows drivers, trace `DriverEntry`, `DriverObject->MajorFunction`, device creation, and IOCTL dispatch. Treat `METHOD_NEITHER`, `FILE_ANY_ACCESS`, unchecked lengths, arbitrary pointer use, and privileged hardware/process operations as leads to verify, not conclusions.
5. Do not call IDA mutating tools (`patch_bytes`, rename, comments, types, variables, data creation, bookmarks) unless the user explicitly asks. Do not execute the target, attach to a live process, or use Perception write/allocate/free operations unless explicitly requested.
6. Finish with a concise evidence table: finding, address, data flow, confidence, and next static check. Keep copied inputs and IDBs local to the working area.

