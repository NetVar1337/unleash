---
name: aida-ida-assistant
description: "AiDA-class IDA Pro AI assistant workflows: function context export, rename/comment proposals, Unreal scans, provider config, review-before-apply."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: re
  triggers:
    - "AiDA"
    - "IDA AI plugin"
    - "IDA 9.4 assistant"
---

# AiDA IDA assistant workflow

For native IDA plugins like **AiDA-Fork** (IDA 9.4): LLM-assisted analysis with human review gates.

## Strengths
- Function Q&A with decompiler context
- Rename/comment/struct proposals
- Hook snippet generation
- Batch analysis + cache
- Unreal global candidate scanning (static evidence only)

## Safe usage rules
- **Review before apply** every DB mutation
- Keep API keys in env/DPAPI/Keychain — never commit
- Treat model output as hypothesis; verify with xrefs/tests
- Pin IDA SDK version to plugin build

## Local path
`Desktop/AiDA-Fork-9.4` / NetVar1337 AiDA-Fork releases

## Pair with
`ida-reverse`, `ida-pro-mcp`, `annotations`, `game-internals`.
