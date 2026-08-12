---
name: systems-language-engineering
description: "Use when engineering Go, Rust, C++23, Java, or Zig projects (layout, build, tests, style)."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: languages
  upstream: C:\Users\Admin\.agents\skills\systems-language-engineering\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\systems-language-engineering\SKILL.md

# Systems Language Engineering Skill

Use the repository's actual build graph and pinned toolchain. This skill routes Go, Rust, C++/C++23, Java, and Zig work to native validation without replacing project instructions or inventing a new build system.

## When to Use

- Editing `.go`, `.rs`, `.c`, `.cc`, `.cpp`, `.cxx`, `.h`, `.hpp`, `.java`, or `.zig` files.
- Reviewing FFI, ABI, ownership, concurrency, unsafe code, templates, modules, or build configuration.
- Establishing a missing local validation loop for one of these languages.

Do not use generic commands blindly in a repository with wrappers or documented gates.

## Prerequisites

Use `terminal` to inspect the manifest and tool versions before editing. Prefer the project's pinned version from `go.mod`, `rust-toolchain.toml`, `CMakePresets.json`, `pom.xml`, Gradle wrapper, or `build.zig.zon` over the machine default.

Expected command families:

| Language | Manifest/build evidence | Primary tools |
|---|---|---|
| Go | `go.mod`, `go.work` | `go`, `gopls`, optional `govulncheck` |
| Rust | `Cargo.toml`, `rust-toolchain.toml` | `cargo`, `rustc`, `rustfmt`, Clippy |
| C++23 | `CMakeLists.txt`, `CMakePresets.json`, Meson/Bazel files | CMake, Ninja, Clang/MSVC/GCC |
| Java | `pom.xml`, `build.gradle*`, wrapper scripts | Maven/Gradle wrapper, JDK |
| Zig | `build.zig`, `build.zig.zon` | `zig`, optional `zls` |

## How to Run

1. Use `search_files` to find repository instructions and manifests.
2. Use `read_file` for the relevant manifest, lockfile, and neighboring implementation.
3. Trace changed symbols to definitions and usages before editing.
4. Make the smallest root-cause change with `patch` or `write_file`.
5. Run the repository-native formatter, static checks, tests, and build through `terminal`.
6. Report the exact commands and their real exit results.

## Quick Reference

### Go

```bash
go version
go env GOMOD GOWORK
gofmt -w <changed-files>
go vet ./...
go test ./...
go test -race ./...
govulncheck ./...
```

Use `go test -race` when concurrency or shared state changed. Preserve `context.Context` cancellation, avoid goroutine leaks, and do not add interfaces before there are multiple consumers or a test seam that needs one.

### Rust

```bash
rustc --version
cargo fmt --all -- --check
cargo clippy --workspace --all-targets --all-features -- -D warnings
cargo test --workspace --all-features
cargo build --workspace --all-features
```

Treat `unsafe` as a local proof obligation: document invariants, minimize its span, and test boundary cases. Do not replace project-selected features with `--all-features` if features are mutually exclusive; follow CI's matrix.

### C++ / C++23

```bash
cmake --preset <configure-preset>
cmake --build --preset <build-preset>
ctest --preset <test-preset> --output-on-failure
clang-tidy <file> -- <compile-flags>
```

If presets do not exist, inspect the documented generator and options before configuring. Preserve RAII, ownership clarity, exception policy, ABI boundaries, and the project's warning level. Use the compile database instead of guessing include paths or defines.

### Java

```bash
./mvnw test
./mvnw verify
./gradlew test
./gradlew check
```

Use the checked-in wrapper. Do not substitute a global Maven or Gradle version. Confirm the configured toolchain/release level before using language features. Keep interrupt status, resource lifetimes, nullability contracts, and module boundaries intact.

### Zig

```bash
zig version
zig fmt --check .
zig build test
zig build
```

Read `build.zig` and `build.zig.zon` before changing dependencies or targets. Make allocator ownership and error unions explicit. Do not assume APIs from another Zig release; the language and standard library evolve quickly.

## Procedure

### 1. Establish the contract

Identify the target, language version, build entry point, CI gate, and observable success criterion. Completion requires one evidence-backed command path, not only editor diagnostics.

### 2. Reproduce or baseline

For a bug, run the narrowest failing test first. For a feature, establish the nearest passing test/build baseline. Record pre-existing failures separately.

### 3. Trace boundaries

Check callers, error paths, serialization/FFI edges, concurrent ownership, and public API compatibility. For generated code, change the source generator rather than generated output unless the repository says otherwise.

### 4. Implement surgically

Prefer existing modules and idioms. Avoid single-use abstractions, dependency additions, broad formatting, or speculative compatibility layers.

### 5. Validate in widening rings

Run: changed-unit test → package/module tests → formatter/static analysis → broader build/test gate. Stop and diagnose the first failure; do not weaken checks to obtain green output.

## Pitfalls

- Running global tools instead of checked-in wrappers or pinned toolchains.
- Assuming the newest language release when the project pins an older one.
- Applying `--all-features` to mutually exclusive Rust features.
- Configuring CMake without the repository's cache variables or presets.
- Using a global Java build tool when wrapper scripts exist.
- Guessing Zig standard-library APIs from memory.
- Reporting editor/LSP cleanliness as proof that the project builds.

## Verification

- [ ] Repository instructions and manifests were read.
- [ ] The active language/toolchain version was identified.
- [ ] Changed symbols and boundary call paths were traced.
- [ ] Formatter and static checks passed on the real code path.
- [ ] Relevant tests passed.
- [ ] The repository-native build passed when the change can affect compilation.
- [ ] Final claims quote real command results and disclose skipped gates.

