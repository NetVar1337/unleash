---
name: pattern-scanner
description: "Binary/pattern scanning: IDA/runtime signatures, mask formats, PE section constraints, SIMD scanners."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: re
---

# Pattern scanner

## Formats
- IDA style `48 8B ?? ?? 75`
- Code + data section limits
- Relative resolve (E8/E9, LEA)

## Quality
- Uniqueness test across modules
- Multi-hit reporting
- Cache by module hash
