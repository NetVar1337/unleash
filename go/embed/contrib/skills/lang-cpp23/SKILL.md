---
name: lang-cpp23
description: "Modern C++23 engineering for systems/game/cheat dev: modules, std ranges, expected, simd, build (CMake/xmake), MSVC/Clang."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: languages
  triggers:
    - "C++23"
    - "cpp23"
    - "modern C++"
---

# C++23 systems/game engineering

## Prefer
- C++23: `std::expected`, `std::print`, ranges/views, `std::mdspan`, improved constexpr
- clang-cl or MSVC latest; `/std:c++23preview` as needed
- warnings-as-errors on core targets; ASan/UBSan in dev

## Game/cheat oriented
- Clear separation: SDK dump bindings vs cheat logic vs render
- Avoid CRT noise in manual maps; know `/GS`, exceptions, TLS constraints in injectors
- Math: your own vec/mat or carefully vendored; force inline hot aim paths

## Project layout
```
src/core  src/game  src/features  src/render  src/driver(optional)
```
