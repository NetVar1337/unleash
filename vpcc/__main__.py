"""
vpcc — Void Patcher for Claude Code
Supports both cli.js (≤2.1.112) and Bun SEA binary (≥2.1.114).
Regex-signature patches survive all minor/patch releases.
"""
from __future__ import annotations
import argparse
import hashlib
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
from datetime import datetime
from pathlib import Path
from typing import Any

from . import updater as _updater
from . import scanner as _scanner

ROOT        = Path(__file__).resolve().parent.parent
PATCH_DIR   = ROOT / "patches"
BACKUP_DIR  = Path.home() / ".vpcc" / "backups"

G, Y, R, B, X = "\033[32m", "\033[33m", "\033[31m", "\033[1m", "\033[0m"

# ── console encoding safety (Windows cp1252) ─────────────────────────────────
# Force UTF-8 on stdout/stderr so Unicode glyphs (✓ ✗ ⚠ →) don't crash the
# process. Falls back to ASCII substitution when reconfigure is unavailable.
try:
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")  # type: ignore[attr-defined]
    sys.stderr.reconfigure(encoding="utf-8", errors="replace")  # type: ignore[attr-defined]
except Exception:
    pass

_enc = (getattr(sys.stdout, "encoding", None) or "").lower()
if "utf" not in _enc:
    # ASCII-only fallbacks for legacy code pages (cp1252, cp437).
    CHECK, CROSS, WARN_ICON, ARROW, DOT = "[ok]", "[X]", "[!]", "->", "*"
    # Kill ANSI colour too — Windows legacy consoles may not render it.
    G = Y = R = B = X = ""
else:
    CHECK, CROSS, WARN_ICON, ARROW, DOT = "✓", "✗", "⚠", "→", "·"

_PKG = "@anthropic-ai/claude-code"
_BUN_SECTION = ".bun"


# ── target discovery ─────────────────────────────────────────────────────────

def _npm_global_roots() -> list[Path]:
    roots: list[Path] = []
    # On Windows the npm shim is npm.cmd (or npm.exe), not bare npm; Python's
    # subprocess.run does not honor PATHEXT, so a bare ["npm", ...] invocation
    # raises FileNotFoundError before the bin resolution ever finds the shim.
    # Probe candidates in order; first success wins.
    npm_candidates = ["npm"]
    if os.name == "nt":
        npm_candidates = ["npm.cmd", "npm.exe", "npm"]
    for npm in npm_candidates:
        try:
            r = subprocess.run([npm, "root", "-g"], capture_output=True, text=True, timeout=5)
            if r.returncode == 0 and r.stdout.strip():
                roots.insert(0, Path(r.stdout.strip()))
                break
        except (FileNotFoundError, OSError):
            continue
        except Exception:
            break
    home = Path.home()
    roots += [
        home / ".npm-global/lib/node_modules",
        home / ".local/lib/node_modules",
        Path("/usr/local/lib/node_modules"),
        Path("/usr/lib/node_modules"),
    ]
    return roots


def _version_glob(base: Path, suffix: str) -> list[Path]:
    import glob as _glob
    return [Path(m) for m in _glob.glob(str(base / "*" / "lib" / "node_modules" / suffix))]


def find_target() -> tuple[Path | None, str]:
    """Return (path, kind) — kind is 'js' or 'bun_sea'.
    Enumerates every CC packaging: legacy cli.js, Linux/macOS/Windows SEA.
    """
    # Primary cli.js (legacy, all OS)
    js_checks  = [(_PKG + "/cli.js", "js")]
    # Direct bin/ wrappers inside the main pkg
    bin_checks = [
        (_PKG + "/bin/claude.exe", "bun_sea"),
        (_PKG + "/bin/claude",     "bun_sea"),
    ]
    # Platform-specific sub-packages (npm optionalDependencies pattern)
    sub_checks = [
        (_PKG + "/node_modules/@anthropic-ai/claude-code-linux-x64/claude",    "bun_sea"),
        (_PKG + "/node_modules/@anthropic-ai/claude-code-linux-arm64/claude",  "bun_sea"),
        (_PKG + "/node_modules/@anthropic-ai/claude-code-darwin-x64/claude",   "bun_sea"),
        (_PKG + "/node_modules/@anthropic-ai/claude-code-darwin-arm64/claude", "bun_sea"),
        (_PKG + "/node_modules/@anthropic-ai/claude-code-win32-x64/claude.exe","bun_sea"),
        (_PKG + "/node_modules/@anthropic-ai/claude-code-win32-arm64/claude.exe","bun_sea"),
    ]
    checks = js_checks + sub_checks + bin_checks

    for npm_root in _npm_global_roots():
        for suffix, kind in checks:
            p = npm_root / suffix
            if p.exists() and (kind == "js" or p.stat().st_size > 1_000_000):
                return p, kind

    # Version-managed Node installs
    mise_base = Path.home() / ".local" / "share" / "mise" / "installs" / "node"
    nvm_base  = Path(os.environ.get("NVM_DIR", str(Path.home() / ".nvm"))) / "versions" / "node"
    fnm_base  = Path(os.environ.get("FNM_DIR", str(Path.home() / ".local" / "share" / "fnm"))) / "node-versions"
    volta_base = Path.home() / ".volta" / "tools" / "image" / "packages"
    # Windows %APPDATA%\npm
    appdata   = os.environ.get("APPDATA")
    extra_bases = [mise_base, nvm_base, fnm_base]
    if appdata:
        extra_bases.append(Path(appdata) / "npm" / "node_modules")
        extra_bases.append(Path(appdata) / "npm")

    for base in extra_bases:
        for suffix, kind in checks:
            # mise / nvm / fnm wrap each Node version in its own dir
            candidates = _version_glob(base, suffix)
            for p in candidates:
                if p.exists() and (kind == "js" or p.stat().st_size > 1_000_000):
                    try:
                        return p.resolve(), kind
                    except OSError:
                        return p, kind

    # Bun global
    bun_global = Path.home() / ".bun" / "install" / "global" / "node_modules"
    for suffix, kind in checks:
        p = bun_global / suffix
        if p.exists() and (kind == "js" or p.stat().st_size > 1_000_000):
            try:
                return p.resolve(), kind
            except OSError:
                return p, kind

    # pnpm global
    pnpm_root: Path | None = None
    try:
        r = subprocess.run(["pnpm", "root", "-g"], capture_output=True, text=True, timeout=5)
        if r.returncode == 0 and r.stdout.strip():
            pnpm_root = Path(r.stdout.strip())
    except Exception:
        pnpm_root = Path.home() / ".local" / "share" / "pnpm" / "global" / "5" / "node_modules"
    if pnpm_root:
        for suffix, kind in checks:
            p = pnpm_root / suffix
            if p.exists() and (kind == "js" or p.stat().st_size > 1_000_000):
                try:
                    return p.resolve(), kind
                except OSError:
                    return p, kind

    # Volta
    for suffix, kind in checks:
        volta_candidates = list(volta_base.glob("@anthropic-ai/claude-code/*/node_modules/" + suffix)) if volta_base.is_dir() else []
        for p in volta_candidates:
            if p.exists() and (kind == "js" or p.stat().st_size > 1_000_000):
                try:
                    return p.resolve(), kind
                except OSError:
                    return p, kind
    for hb in [Path("/opt/homebrew/lib/node_modules"),
               Path("/usr/local/lib/node_modules")]:
        for suffix, kind in checks:
            p = hb / suffix
            if p.exists() and (kind == "js" or p.stat().st_size > 1_000_000):
                try:
                    return p.resolve(), kind
                except OSError:
                    return p, kind

    # Native CLI installer (~/.local/share/claude/versions/<ver>)
    _versions_dir = Path.home() / ".local" / "share" / "claude" / "versions"
    if _versions_dir.is_dir():
        for child in sorted(_versions_dir.iterdir(), key=lambda p: p.name, reverse=True):
            if child.is_file() and child.stat().st_size > 1_000_000:
                try:
                    return child.resolve(), "bun_sea"
                except OSError:
                    return child, "bun_sea"

    # Native installer candidate paths (symlinks, legacy layouts)
    native_candidates = [
        Path.home() / ".local" / "bin" / "claude",
        Path.home() / ".local" / "bin" / "claude.exe",
        Path.home() / ".claude" / "local" / "claude",
        Path.home() / ".claude" / "local" / "claude.exe",
        Path.home() / ".claude" / "bin" / "claude",
        Path.home() / ".claude" / "bin" / "claude.exe",
        Path.home() / ".local" / "share" / "claude-code" / "claude",
        Path.home() / ".local" / "share" / "claude-code" / "claude.exe",
        Path.home() / ".local" / "bin" / "claude-code",
        Path.home() / ".local" / "bin" / "claude-code.exe",
        Path("/usr/local/share/claude-code/claude"),
        Path("/opt/claude-code/bin/claude"),
    ]
    for p in native_candidates:
        if p.is_dir():
            for child in sorted(p.glob("claude-*"), reverse=True):
                if child.is_file() and child.stat().st_size > 1_000_000:
                    try:
                        return child.resolve(), "bun_sea"
                    except OSError:
                        return child, "bun_sea"
            continue
        if p.exists() and p.stat().st_size > 1_000_000:
            try:
                return p.resolve(), "bun_sea"
            except OSError:
                return p, "bun_sea"

    # Windows %LOCALAPPDATA%\Programs\claude-code\ (installer default on Win)
    la = os.environ.get("LOCALAPPDATA")
    if la:
        for suffix in ("Programs/claude-code/claude.exe",
                       "Programs/claude/versions",
                       "claude-code/claude.exe",
                       "anthropic/claude-code/claude.exe"):
            p = Path(la) / suffix
            if p.is_dir():
                # e.g. Programs/claude/versions/* — pick newest binary
                for child in sorted(p.iterdir(), reverse=True):
                    if child.is_file() and child.stat().st_size > 1_000_000:
                        try:
                            return child.resolve(), "bun_sea"
                        except OSError:
                            return child, "bun_sea"
            elif p.exists() and p.stat().st_size > 1_000_000:
                try:
                    return p.resolve(), "bun_sea"
                except OSError:
                    return p, "bun_sea"
        # winget
        winget_dir = Path(la) / "Microsoft" / "WinGet" / "Packages"
        if winget_dir.is_dir():
            for pkg_dir in winget_dir.iterdir():
                if "claude" in pkg_dir.name.lower() and pkg_dir.is_dir():
                    for f in pkg_dir.rglob("claude.exe"):
                        if f.is_file() and f.stat().st_size > 1_000_000:
                            try:
                                return f.resolve(), "bun_sea"
                            except OSError:
                                return f, "bun_sea"

    # scoop (Windows)
    scoop_dir = Path.home() / "scoop" / "apps" / "claude-code" / "current"
    if scoop_dir.is_dir():
        for name in ("claude.exe", "claude"):
            p = scoop_dir / name
            if p.exists() and p.stat().st_size > 1_000_000:
                try:
                    return p.resolve(), "bun_sea"
                except OSError:
                    return p, "bun_sea"

    # chocolatey (Windows)
    choco_dir = Path("C:/ProgramData/chocolatey/lib/claude-code")
    if choco_dir.is_dir():
        for f in choco_dir.rglob("claude.exe"):
            if f.is_file() and f.stat().st_size > 1_000_000:
                try:
                    return f.resolve(), "bun_sea"
                except OSError:
                    return f, "bun_sea"

    # Last resort: shutil.which (cross-platform, no `which` dependency)
    try:
        found = shutil.which("claude") or shutil.which("claude.exe")
        if found:
            p = Path(found)
            try:
                p = p.resolve()
            except OSError:
                pass
            if p.exists() and p.stat().st_size > 1_000_000:
                return p, "bun_sea"
    except Exception:
        pass
    return None, ""


def sha256_short(path: Path) -> str:
    # Use C-level streaming hash on py>=3.11 (zero-copy, ~3x faster on big binaries).
    try:
        with path.open("rb") as f:
            return hashlib.file_digest(f, "sha256").hexdigest()[:12]
    except (AttributeError, TypeError):
        h = hashlib.sha256()
        with path.open("rb") as f:
            for chunk in iter(lambda: f.read(1 << 20), b""):
                h.update(chunk)
        return h.hexdigest()[:12]


# ── Bun SEA helpers ───────────────────────────────────────────────────────────

import struct as _struct

_BUN_TRAILER = b"\n---- Bun! ----\n"


def _bun_section_has_valid_trailer(data: bytes | bytearray, bun_lo: int, bun_hi: int) -> bool:
    """Accept Bun section trailer with optional PE file-alignment NUL padding."""
    section = bytes(data[bun_lo:bun_hi]).rstrip(b"\x00")
    return section.endswith(_BUN_TRAILER)


def _bun_section_is_bytecode(section_bytes: bytes) -> bool:
    """True when .bun section uses compiled Bun bytecode — must use in-place patching."""
    return b"// @bun @bytecode" in section_bytes[:1024]


def _find_bun_section_elf(data) -> tuple[int, int]:
    """Linux ELF — shdr walk for .bun section."""
    e_shoff     = _struct.unpack_from("<Q", data, 0x28)[0]
    e_shentsize = _struct.unpack_from("<H", data, 0x3A)[0]
    e_shnum     = _struct.unpack_from("<H", data, 0x3C)[0]
    e_shstrndx  = _struct.unpack_from("<H", data, 0x3E)[0]
    strtab_shdr = e_shoff + e_shstrndx * e_shentsize
    strtab_off  = _struct.unpack_from("<Q", data, strtab_shdr + 0x18)[0]
    strtab_size = _struct.unpack_from("<Q", data, strtab_shdr + 0x20)[0]
    strtab      = bytes(data[strtab_off:strtab_off + strtab_size])
    for i in range(e_shnum):
        sh = e_shoff + i * e_shentsize
        sh_name = _struct.unpack_from("<I", data, sh)[0]
        end = strtab.index(b"\x00", sh_name)
        if strtab[sh_name:end] == b".bun":
            return (_struct.unpack_from("<Q", data, sh + 0x18)[0],
                    _struct.unpack_from("<Q", data, sh + 0x20)[0])
    raise RuntimeError(".bun section not found (ELF)")


def _find_bun_section_macho(data) -> tuple[int, int]:
    """
    macOS Mach-O (x64 + arm64, both endian + fat).
    Walks LC_SEGMENT_64 load commands looking for section named `__bun`
    (segname `__BUN` — Bun's SEA embedding convention).
    """
    magic = _struct.unpack_from(">I", data, 0)[0]
    # Fat binary — pick first 64-bit arch entry.
    if magic in (0xCAFEBABE, 0xBEBAFECA, 0xCAFEBABF, 0xBFBAFECA):
        big = magic in (0xCAFEBABE, 0xCAFEBABF)
        fmt = ">I" if big else "<I"
        nfat = _struct.unpack_from(fmt, data, 4)[0]
        # each fat_arch is 20 bytes (32-bit) or 32 bytes (64-bit)
        entry_size = 20 if magic in (0xCAFEBABE, 0xBEBAFECA) else 32
        for i in range(nfat):
            base = 8 + i * entry_size
            off = _struct.unpack_from(fmt, data, base + 8)[0]
            sub = data[off:] if hasattr(data, "__getitem__") else bytes(data)[off:]
            sub_off, sub_size = _find_bun_section_macho(bytes(sub))
            return (off + sub_off, sub_size)  # Adjust offset relative to original data
    if magic not in (0xFEEDFACF, 0xCFFAEDFE, 0xFEEDFACE, 0xCEFAEDFE):
        raise RuntimeError("not a Mach-O binary")
    is_64 = magic in (0xFEEDFACF, 0xCFFAEDFE)
    be = magic in (0xFEEDFACF, 0xFEEDFACE)
    end = ">" if be else "<"
    # header: magic(4) cputype(4) cpusubtype(4) filetype(4) ncmds(4) sizeofcmds(4) flags(4) [reserved(4) if 64]
    ncmds = _struct.unpack_from(end + "I", data, 16)[0]
    hdr_size = 32 if is_64 else 28
    cur = hdr_size
    LC_SEGMENT_64 = 0x19
    LC_SEGMENT    = 0x01
    for _ in range(ncmds):
        cmd, cmdsize = _struct.unpack_from(end + "II", data, cur)
        if cmd == LC_SEGMENT_64:
            # segment_command_64: cmd(4) cmdsize(4) segname(16) vmaddr(8) vmsize(8) fileoff(8) filesize(8) maxprot(4) initprot(4) nsects(4) flags(4) = 72
            segname = bytes(data[cur+8:cur+24]).split(b"\x00", 1)[0]
            nsects  = _struct.unpack_from(end + "I", data, cur + 64)[0]
            sect_base = cur + 72
            for j in range(nsects):
                # section_64: sectname(16) segname(16) addr(8) size(8) offset(4) align(4) reloff(4) nreloc(4) flags(4) reserved1(4) reserved2(4) reserved3(4) = 80
                s = sect_base + j * 80
                sectname = bytes(data[s:s+16]).split(b"\x00", 1)[0]
                sect_segname = bytes(data[s+16:s+32]).split(b"\x00", 1)[0]
                if sectname in (b"__bun", b".bun") or sect_segname in (b"__BUN",):
                    size   = _struct.unpack_from(end + "Q", data, s + 40)[0]
                    offset = _struct.unpack_from(end + "I", data, s + 48)[0]
                    return (offset, size)
        elif cmd == LC_SEGMENT:
            segname = bytes(data[cur+8:cur+24]).split(b"\x00", 1)[0]
            nsects  = _struct.unpack_from(end + "I", data, cur + 48)[0]
            sect_base = cur + 56
            for j in range(nsects):
                s = sect_base + j * 68
                sectname = bytes(data[s:s+16]).split(b"\x00", 1)[0]
                if sectname in (b"__bun", b".bun"):
                    size   = _struct.unpack_from(end + "I", data, s + 36)[0]
                    offset = _struct.unpack_from(end + "I", data, s + 40)[0]
                    return (offset, size)
        cur += cmdsize
    raise RuntimeError(".bun/__bun section not found (Mach-O)")


def _find_bun_section_pe(data) -> tuple[int, int]:
    """Windows PE/COFF — walks section table for .bun section."""
    if data[:2] != b"MZ":
        raise RuntimeError("not a PE binary")
    pe_off = _struct.unpack_from("<I", data, 0x3C)[0]
    if bytes(data[pe_off:pe_off+4]) != b"PE\x00\x00":
        raise RuntimeError("PE signature missing")
    coff = pe_off + 4
    nsect = _struct.unpack_from("<H", data, coff + 2)[0]
    opt_size = _struct.unpack_from("<H", data, coff + 16)[0]
    sect_table = coff + 20 + opt_size
    # each IMAGE_SECTION_HEADER = 40 bytes
    for i in range(nsect):
        s = sect_table + i * 40
        name = bytes(data[s:s+8]).split(b"\x00", 1)[0]
        if name == b".bun":
            vsize   = _struct.unpack_from("<I", data, s + 8)[0]
            rsize   = _struct.unpack_from("<I", data, s + 16)[0]
            raw_off = _struct.unpack_from("<I", data, s + 20)[0]
            # PE SizeOfRawData includes file-alignment padding. VirtualSize is
            # the live .bun payload length; using raw size can land trailer
            # validation inside trailing NUL padding on Windows builds.
            return (raw_off, vsize or rsize)
    raise RuntimeError(".bun section not found (PE)")


def _find_bun_section(data) -> tuple[int, int]:
    """Dispatch to ELF / Mach-O / PE parser by magic bytes. Cross-OS."""
    head = bytes(data[:4])
    if head[:4] == b"\x7fELF":
        return _find_bun_section_elf(data)
    if head in (b"\xcf\xfa\xed\xfe", b"\xfe\xed\xfa\xcf",
                b"\xce\xfa\xed\xfe", b"\xfe\xed\xfa\xce",
                b"\xca\xfe\xba\xbe", b"\xbe\xba\xfe\xca",
                b"\xca\xfe\xba\xbf", b"\xbf\xba\xfe\xca"):
        return _find_bun_section_macho(data)
    if head[:2] == b"MZ":
        return _find_bun_section_pe(data)
    raise RuntimeError(f"unknown binary format magic={head.hex()}")


def _find_active_bundle_bounds(data, bun_lo: int, bun_hi: int) -> tuple[int, int]:
    """
    Bun SEA ≥ 2.1.119 embeds the main entry bundle PLUS a virtual-filesystem
    (VFS) copy of the same source at a second location.  The layout is:

        [header][active-entry-blob (JS source, ~13 MB)][bytecode-for-deps (~102 MB)]
        [lookup-key\x00][vfs-path\x00][vfs-copy-of-cli.js][other-vfs-files][trailer]

    Patching any byte in the VFS copy corrupts the module Bun re-parses at
    startup, causing the "entry.instantiate" crash even though the active
    bundle itself is fine.

    Detection: the active bundle always starts with "// @bun @bytecode" and its
    byte-length is encoded as a little-endian u32 at (data_start - 4).
    We read that field and return [abs_start, abs_end) so callers only scan the
    active bundle region.

    Falls back to the full [bun_lo, bun_hi) range if the marker or size look
    implausible (e.g., old single-bundle builds).
    """
    BUN_BYTECODE_MARKER = b"// @bun @bytecode"
    section_start = bun_lo
    # Search for the marker within the section
    abs_marker = data.find(BUN_BYTECODE_MARKER, section_start, bun_hi)
    if abs_marker < 0:
        return bun_lo, bun_hi  # fallback: no marker found

    data_start = abs_marker  # absolute offset in `data`
    size_field_off = data_start - 4
    if size_field_off < section_start:
        return bun_lo, bun_hi  # no room for size field

    import struct as _s
    blob_size = _s.unpack_from("<I", data, size_field_off)[0]
    active_end = data_start + blob_size

    # Sanity: blob_size must be positive and fit inside the section
    if blob_size < 1024 or active_end > bun_hi:
        return bun_lo, bun_hi  # implausible — fall back

    return data_start, active_end


def patch_bun_sea_inplace(binary: Path, patches: list) -> dict:
    """
    In-place Bun SEA byte patcher. No objcopy, preserves ELF layout.
    Runs the patched binary to verify before committing.

    Research ref: docs/BUN_BYTECODE_FORMAT.md in void-patcher repo.
    Key insight: .bun section has no integrity check; JSC SourceCodeKey is
    fail-open (mismatch -> bytecode discarded, source re-parsed, app runs).

    2.1.119 change: the .bun section now contains a second VFS copy of the
    main cli.js source.  Patching that copy corrupts Bun's module-loader.
    _find_active_bundle_bounds() restricts writes to the active-entry blob only.
    """
    mode = binary.stat().st_mode & 0o7777
    original_size = binary.stat().st_size
    data = bytearray(binary.read_bytes())

    bun_off, bun_size = _find_bun_section(data)
    bun_lo, bun_hi = bun_off, bun_off + bun_size

    if not _bun_section_has_valid_trailer(data, bun_lo, bun_hi):
        return {"ok": False, "err": "Bun trailer invalid — format change", "applied": 0, "skipped": 0}

    # Narrow the writable region to the active entry bundle only, so we never
    # accidentally corrupt the VFS copy or the bundled-module bytecode blobs.
    eff_lo, eff_hi = _find_active_bundle_bounds(data, bun_lo, bun_hi)

    applied_total = 0
    skipped_total = 0
    per_patch = []

    for p in patches:
        applied_n = 0
        skipped_n = 0
        max_padding = 0
        for sub in p.get("patches", []):
            search_regex = sub.get("search_regex")
            search = sub.get("search")
            replace = sub.get("replace", "")
            marker = sub.get("applied_marker")

            if marker and data.find(marker.encode("utf-8", "surrogateescape"), eff_lo, eff_hi) >= 0:
                continue

            if search_regex:
                try:
                    pat = re.compile(search_regex.encode("utf-8", "surrogateescape"), re.DOTALL)
                except re.error:
                    skipped_n += 1
                    continue
                section_view = bytes(data[eff_lo:eff_hi])
                for m in pat.finditer(section_view):
                    mb = m.group(0)
                    try:
                        rb = m.expand(replace.encode("utf-8", "surrogateescape"))
                    except Exception:
                        skipped_n += 1
                        continue
                    if len(rb) > len(mb):
                        skipped_n += 1
                        continue
                    if len(rb) < len(mb):
                        padding = len(mb) - len(rb)
                        rb = rb + b" " * padding
                        if padding > max_padding:
                            max_padding = padding
                    abs_start = eff_lo + m.start()
                    data[abs_start:abs_start + len(mb)] = rb
                    applied_n += 1
            elif search:
                s_b = search.encode("utf-8", "surrogateescape")
                r_b = replace.encode("utf-8", "surrogateescape")
                if len(r_b) > len(s_b):
                    skipped_n += 1
                    continue
                if len(r_b) < len(s_b):
                    padding = len(s_b) - len(r_b)
                    r_b = r_b + b" " * padding
                    if padding > max_padding:
                        max_padding = padding
                pos = eff_lo
                while True:
                    j = data.find(s_b, pos, eff_hi)
                    if j < 0:
                        break
                    data[j:j + len(s_b)] = r_b
                    applied_n += 1
                    pos = j + len(s_b)

        per_patch.append({"id": p["id"], "applied": applied_n, "skipped": skipped_n, "max_padding": max_padding})
        applied_total += applied_n
        skipped_total += skipped_n

    if len(data) != original_size:
        return {"ok": False, "err": f"size drift {len(data)} vs {original_size}",
                "applied": 0, "skipped": skipped_total, "per_patch": per_patch}

    if applied_total == 0:
        return {"ok": True, "noop": True, "applied": 0, "skipped": skipped_total, "per_patch": per_patch}

    # Write to temp same-dir file, verify by running, then atomic swap.
    tmp_bin = binary.parent / f".{binary.name}.vpcctmp-{os.getpid()}"
    try:
        tmp_bin.write_bytes(bytes(data))
        try:
            tmp_bin.chmod(mode)
        except OSError:
            pass  # Windows: chmod not fully supported

        # macOS: re-sign with ad-hoc signature after patching (code signature
        # invalidated by byte changes; unsigned Mach-O gets SIGKILL on arm64).
        if sys.platform == "darwin":
            subprocess.run(["codesign", "--force", "--sign", "-", str(tmp_bin)],
                           capture_output=True, timeout=30)

        r = subprocess.run([str(tmp_bin), "--version"],
                           capture_output=True, text=True, timeout=60)
        out = (r.stdout or "") + (r.stderr or "")
        if r.returncode != 0 or "Claude Code" not in out:
            tmp_bin.unlink(missing_ok=True)
            return {"ok": False, "err": f"verify failed: {out[:120]!r} rc={r.returncode}",
                    "applied": applied_total, "skipped": skipped_total, "per_patch": per_patch}

        os.replace(str(tmp_bin), str(binary))
        try:
            binary.chmod(mode)
        except OSError:
            pass  # Windows: chmod not fully supported
    except Exception as e:
        tmp_bin.unlink(missing_ok=True)
        return {"ok": False, "err": f"write failed: {e}", "applied": 0, "skipped": skipped_total}

    return {"ok": True, "applied": applied_total, "skipped": skipped_total, "per_patch": per_patch}


def read_bun_js(binary: Path) -> tuple[str | None, str]:
    """Extract JS text from .bun ELF section. Copies binary first (ETXTBSY guard)."""
    with tempfile.TemporaryDirectory() as tmp:
        src = Path(tmp) / "src.exe"
        shutil.copy2(binary, src)
        section = Path(tmp) / "section.bin"
        r = subprocess.run(
            ["objcopy", "--dump-section", f"{_BUN_SECTION}={section}", str(src)],
            capture_output=True, text=True,
        )
        if r.returncode != 0:
            return None, f"objcopy dump: {(r.stderr or r.stdout).strip()}"
        data = section.read_bytes()
        if _bun_section_is_bytecode(data):
            return None, "Bun bytecode format — text patching not supported (would corrupt binary)"
        try:
            return data.decode("utf-8", errors="surrogateescape"), ""
        except Exception as e:
            return None, str(e)


def write_bun_js(binary: Path, text: str) -> tuple[bool, str]:
    """HARD LOCK: Bun SEA bytecode cannot be safely text-patched. Refuse writes."""
    return False, "Bun SEA bytecode — binary writes disabled to prevent corruption"

def _write_bun_js_DISABLED(binary: Path, text: str) -> tuple[bool, str]:
    """Inject patched JS text back into .bun ELF section. Unlinks first (ETXTBSY guard)."""
    with tempfile.TemporaryDirectory() as tmp:
        src = Path(tmp) / "src.exe"
        shutil.copy2(binary, src)
        section = Path(tmp) / "section.bin"
        section.write_bytes(text.encode("utf-8", errors="surrogateescape"))
        out = Path(tmp) / "patched.exe"
        shutil.copy2(src, out)
        r = subprocess.run(
            ["objcopy", "--update-section", f"{_BUN_SECTION}={section}", str(out)],
            capture_output=True, text=True,
        )
        if r.returncode != 0:
            return False, f"objcopy inject: {(r.stderr or r.stdout).strip()}"
        mode = binary.stat().st_mode & 0o7777
        binary.unlink()        # must unlink running binary before replacing (ETXTBSY)
        shutil.copy2(out, binary)
        binary.chmod(mode)
    return True, ""


# ── patch logic ───────────────────────────────────────────────────────────────

def _is_retired_json(f: Path) -> bool:
    """Quick check: is this patch JSON marked retired?"""
    try:
        return bool(json.loads(f.read_text(encoding="utf-8")).get("retired"))
    except (json.JSONDecodeError, OSError):
        return False

def load_patches() -> list[dict[str, Any]]:
    patches = []
    for f in sorted(PATCH_DIR.glob("*.json")):
        try:
            p = json.loads(f.read_text(encoding="utf-8"))
            if p.get("disabled"):
                continue
            if p.get("retired"):
                continue
            patches.append(p)
        except json.JSONDecodeError as e:
            print(f"{R}{CROSS} {f.name}: invalid JSON — {e}{X}", file=sys.stderr)
    return patches


def _apply_subs(text: str, patch: dict) -> tuple[str, int, str]:
    """Apply sub-patch list to text. Returns (new_text, n_applied, error)."""
    total = 0
    for sub in patch.get("patches", []):
        rep    = sub.get("replace", "")
        marker = sub.get("applied_marker")
        if marker and marker in text:
            continue  # idempotent — already applied

        pat      = sub.get("search_regex") or sub.get("search")
        is_regex = bool(sub.get("search_regex"))
        if not pat:
            continue
        try:
            if is_regex:
                new_text, n = re.subn(pat, rep, text,
                                      count=sub.get("count", 0) or 0,
                                      flags=re.DOTALL)
            else:
                cnt = sub.get("count", -1)
                new_text = text.replace(pat, rep, cnt if cnt > 0 else -1)
                n = int(new_text != text)
        except re.error as e:
            return text, total, f"regex error: {e}"

        if n == 0 and sub.get("required", False):
            return text, total, f"required pattern not found: {pat[:60]}..."
        text = new_text
        total += n if isinstance(n, int) else int(n)

    return text, total, ""


def backup(target: Path, kind: str) -> Path:
    BACKUP_DIR.mkdir(parents=True, exist_ok=True)
    stamp = datetime.now().strftime("%Y%m%d-%H%M%S")
    ext   = "exe.bak" if kind == "bun_sea" else "js.bak"
    dst   = BACKUP_DIR / f"claude.{stamp}.{sha256_short(target)}.{ext}"
    shutil.copy2(target, dst)
    for old in sorted(BACKUP_DIR.glob("claude.*.*"))[:-10]:
        old.unlink(missing_ok=True)
    return dst


# ── commands ──────────────────────────────────────────────────────────────────

def cmd_patch(args) -> int:
    patches  = load_patches()
    target, kind = find_target()
    js_patches   = [p for p in patches if p.get("type") == "js_replace"]
    meta_patches = [p for p in patches if p.get("type") not in ("js_replace",)]

    print(f"{B}vpcc patch — {len(patches)} patches{X}")
    if target:
        label = "cli.js" if kind == "js" else f"Bun SEA {target.name}"
        print(f"  target : {target}")
        print(f"  format : {label}")
        print(f"  sha    : {sha256_short(target)}")
        if not args.dry_run:
            bkp = backup(target, kind)
            print(f"  backup : {bkp}")
    else:
        print(f"  {Y}target not found — settings-only patches will apply{X}")

    ok = fail = skip = 0

    # ── JS patches ─────────────────────────────────────────────────────────
    if not target:
        for p in js_patches:
            print(f"  {Y}skip{X} {p['id']:40s}  target not found")
            skip += 1
    elif kind == "bun_sea":
        # In-place ELF byte patcher — no objcopy, no corruption. Applies every
        # js_replace in a single pass, verifies by running patched binary, atomic swap.
        if args.dry_run:
            # Dry-run: apply to a temporary in-memory bytearray, count hits only.
            try:
                data = bytearray(target.read_bytes())
                bun_off, bun_size = _find_bun_section(data)
                eff_lo, eff_hi = _find_active_bundle_bounds(data, bun_off, bun_off + bun_size)
                for p in js_patches:
                    applied_n = 0
                    for sub in p.get("patches", []):
                        marker = sub.get("applied_marker")
                        if marker and data.find(marker.encode("utf-8","surrogateescape"), eff_lo, eff_hi) >= 0:
                            continue
                        sr = sub.get("search_regex") or sub.get("search") or ""
                        if not sr:
                            continue
                        try:
                            if sub.get("search_regex"):
                                pat = re.compile(sr.encode("utf-8","surrogateescape"), re.DOTALL)
                                applied_n += sum(1 for _ in pat.finditer(bytes(data[eff_lo:eff_hi])))
                            else:
                                applied_n += bytes(data[eff_lo:eff_hi]).count(sr.encode("utf-8","surrogateescape"))
                        except re.error:
                            pass
                    msg = "no-op (already applied)" if applied_n == 0 else f"would apply {applied_n} in-place"
                    print(f"  {G}ok{X}    {p['id']:40s}  {msg}")
                    ok += 1
                print(f"  {Y}dry-run: binary not modified{X}")
            except Exception as e:
                print(f"  {R}fail{X}  [bun-inplace]  {e}")
                fail += len(js_patches)
        else:
            result = patch_bun_sea_inplace(target, js_patches)
            if not result["ok"]:
                print(f"  {R}fail{X}  [bun-inplace]  {result.get('err','unknown')}")
                fail += len(js_patches)
                # Patches that needed the most padding are the most likely corruption source.
                heavy = sorted(
                    [pr for pr in result.get("per_patch", []) if pr.get("max_padding", 0) > 64],
                    key=lambda x: -x["max_padding"],
                )
                if heavy:
                    print(f"  {Y}hint: {len(heavy)} patch(es) needed >64 bytes padding — "
                          f"likely corruption source:{X}")
                    for pr in heavy[:5]:
                        print(f"    {pr['id']}: {pr['max_padding']} bytes padding needed")
                    print(f"  {Y}run 'vpcc scan' for signature analysis or 'vpcc rollback' to revert{X}")
            else:
                for pr in result.get("per_patch", []):
                    n = pr["applied"]
                    msg = "no-op (already applied)" if n == 0 else f"{n} in-place replacement(s)"
                    print(f"  {G}ok{X}    {pr['id']:40s}  {msg}")
                    ok += 1
                if result.get("noop"):
                    pass  # all already applied
                else:
                    print(f"  {G}verified in-place{X}  (ran binary, Claude Code output confirmed)")
    else:
        # cli.js path — patch file directly
        text = target.read_text(encoding="utf-8")
        orig = text
        for p in js_patches:
            new_text, n, err = _apply_subs(text, p)
            if err:
                print(f"  {R}fail{X}  {p['id']:40s}  {err}")
                fail += 1
                continue
            msg = "no-op (already applied)" if new_text == text else f"{n} replacement(s)"
            print(f"  {G}ok{X}    {p['id']:40s}  {msg}")
            text = new_text
            ok  += 1
        if text != orig and not args.dry_run:
            # Atomic write: stage to sibling tmp, node --check it, only then
            # replace the live cli.js. On any failure the original is untouched.
            tmp = target.parent / f".{target.name}.vpcctmp-{os.getpid()}"
            try:
                tmp.write_text(text, encoding="utf-8")
                r = subprocess.run(["node", "--check", str(tmp)],
                                   capture_output=True, text=True, timeout=15)
                if r.returncode != 0:
                    tmp.unlink(missing_ok=True)
                    err = (r.stderr or r.stdout or "").strip().splitlines()[-1:]
                    print(f"\n{R}{CROSS} cli.js syntax INVALID — aborted, original untouched{X}")
                    if err:
                        print(f"  node: {err[0][:180]}")
                    return 2
                try:
                    shutil.copystat(target, tmp)
                except Exception:
                    pass
                os.replace(tmp, target)   # atomic on same filesystem
            except Exception as e:
                tmp.unlink(missing_ok=True)
                print(f"\n{R}{CROSS} write failed: {e} — cli.js untouched{X}")
                return 2

    # ── non-JS patches ─────────────────────────────────────────────────────
    for p in meta_patches:
        t = p.get("type")
        if t in ("settings", "hook"):
            success, msg = _apply_settings(p, dry_run=args.dry_run)
        elif t == "mcp_guard":
            success, msg = _apply_mcp_guard(p, dry_run=args.dry_run)
        elif t == "wrapper":
            success, msg = _apply_wrapper(p, kind or "bun_sea", target, dry_run=args.dry_run)
        elif t == "binary_install":
            success, msg = _apply_binary_install(p, kind or "bun_sea", dry_run=args.dry_run)
        else:
            print(f"  {Y}skip{X} {p['id']:40s}  type={t} (unknown)")
            skip += 1
            continue
        mark = f"{G}ok{X}" if success else f"{R}fail{X}"
        print(f"  {mark}   {p['id']:40s}  {msg}")
        ok += success
        fail += not success

    print(f"\n{B}{ok} ok {DOT} {fail} failed {DOT} {skip} skipped{X}")

    # record state for autoheal drift detection
    if target and not args.dry_run and not fail:
        try:
            _updater.save_state(last_cc_sha=sha256_short(target), last_cc_kind=kind)
        except Exception:
            pass

        # Stamp SHA for guard fast-path
        try:
            stamp = Path.home() / ".vpcc" / "last_patched_sha"
            stamp.parent.mkdir(parents=True, exist_ok=True)
            stamp.write_text(sha256_short(target))
        except OSError:
            pass

    return 1 if fail else 0


def _apply_settings(patch: dict, dry_run: bool = False) -> tuple[bool, str]:
    custom = patch.get("settings_path")
    if custom:
        path = Path(os.path.expanduser(custom))
    else:
        path = Path.home() / ".claude" / "settings.json"
        if not path.exists() and os.name == 'nt':
            _appdata = os.environ.get("APPDATA")
            if _appdata:
                alt = Path(_appdata) / "claude" / "settings.json"
                if alt.exists():
                    path = alt
    path.parent.mkdir(parents=True, exist_ok=True)
    try:
        cur = json.loads(path.read_text(encoding="utf-8")) if path.is_file() else {}
    except json.JSONDecodeError:
        cur = {}
    wanted = patch.get("settings", {})
    out = dict(cur)
    changed = sum(1 for k, v in wanted.items() if out.setdefault(k, None) != v and not out.__setitem__(k, v))  # type: ignore
    # simpler:
    out = dict(cur)
    changed = 0
    for k, v in wanted.items():
        if out.get(k) != v:
            out[k] = v
            changed += 1
    if not changed:
        return True, "no-op (already applied)"
    if dry_run:
        return True, f"would set {changed} key(s)"
    path.write_text(json.dumps(out, indent=2), encoding="utf-8")
    return True, f"{changed} key(s)"


def _apply_mcp_guard(p: dict, dry_run: bool = False) -> tuple[bool, str]:
    """mcp-guard: add MCP spawn timeout to preload + activate via BUN_OPTIONS wrapper."""
    if os.name == 'nt':
        preload_dir = Path(os.environ.get("LOCALAPPDATA", str(Path.home() / "AppData" / "Local"))) / "void-patcher"
    else:
        preload_dir = Path.home() / ".local" / "share" / "void-patcher"
    preload_src = ROOT / "contrib" / "preload" / "claude-preload.js"
    preload_dst = preload_dir / "claude-preload.js"
    GUARD_MARKER = "// mcp-guard"
    GUARD_CODE = (
        "\n  // mcp-guard: timeout npx @latest MCP server spawns\n"
        '  try { const _cp = require("child_process"); const _oS = _cp.spawn;\n'
        "    _cp.spawn = function(cmd, args, opts) { const p = _oS.apply(this, arguments);\n"
        '      const isNpx = typeof cmd === "string" && cmd.includes("npx") &&\n'
        '        Array.isArray(args) && args.some(a => typeof a === "string" && a.includes("@latest"));\n'
        '      if (isNpx) { const t = setTimeout(() => { try { p.kill(); } catch(_) {} }, 30000);\n'
        '        p.on("close", () => clearTimeout(t)); } return p; }; } catch(_) {}\n'
    )

    def inject_guard(content: str) -> str | None:
        inject_at = "globalThis.__VPCC_PRELOAD_ACTIVE__ = true;"
        if inject_at in content:
            return content.replace(inject_at, inject_at + GUARD_CODE, 1)
        tail = "\n})();"
        idx = content.rfind(tail)
        if idx >= 0:
            return content[:idx] + GUARD_CODE + content[idx:]
        return None

    preload_missing = not preload_dst.exists()
    if preload_missing:
        if not preload_src.exists():
            return False, "preload source missing — run vpcc install-preload first"
        content = preload_src.read_text(encoding="utf-8")
        if not dry_run:
            preload_dir.mkdir(parents=True, exist_ok=True)
            preload_dst.write_text(content, encoding="utf-8")
    else:
        content = preload_dst.read_text(encoding="utf-8")

    already_guarded = GUARD_MARKER in content
    if already_guarded:
        if dry_run and preload_missing:
            return True, "would install preload (mcp-guard already present)"
        return True, "no-op (already applied)"

    new_content = inject_guard(content)
    if new_content is None:
        return False, "preload format unexpected"

    if dry_run:
        if preload_missing:
            return True, "would install preload and add mcp-guard"
        return True, "would add mcp-guard to preload"

    preload_dst.write_text(new_content, encoding="utf-8")
    return True, "mcp-guard added to preload"


def _apply_wrapper(p: dict, kind: str, target: "Path | None", dry_run: bool = False) -> tuple[bool, str]:
    """Install wrapper — native binary or cli.js depending on install kind."""
    wrapper_path = Path(p.get("wrapper_path", "~/.local/bin/claude")).expanduser()
    NATIVE_MARKER = "# vpcc-native-wrapper"
    JS_MARKER = "CLAUDE_CLI_JS"

    if kind == "bun_sea":
        WRAPPER_CONTENT = (
            "#!/usr/bin/env bash\n"
            f"{NATIVE_MARKER}\n"
            'VPCC_PRELOAD="${VPCC_PRELOAD:-$HOME/.local/share/void-patcher/claude-preload.js}"\n'
            'VPCC_VERSIONS_DIR="$HOME/.local/share/claude/versions"\n'
            'VPCC_NATIVE="$(ls -1 "$VPCC_VERSIONS_DIR" 2>/dev/null | sort -t. -k1,1n -k2,2n -k3,3n | tail -1)"\n'
            'VPCC_NATIVE="$VPCC_VERSIONS_DIR/$VPCC_NATIVE"\n'
            'if [[ ! -x "${VPCC_NATIVE:-}" ]]; then echo "vpcc wrapper: native binary not found" >&2; exit 1; fi\n'
            'if [[ -f "$VPCC_PRELOAD" ]]; then\n'
            '    export BUN_OPTIONS="${BUN_OPTIONS:+$BUN_OPTIONS }--preload $VPCC_PRELOAD"\n'
            'fi\n'
            'exec "$VPCC_NATIVE" "$@"\n'
        )
        if wrapper_path.exists() and not wrapper_path.is_symlink():
            try:
                if NATIVE_MARKER in wrapper_path.read_text():
                    return True, "no-op (already applied)"
            except Exception:
                pass

        if dry_run:
            return True, "would install native BUN_OPTIONS wrapper"

        wrapper_path.parent.mkdir(parents=True, exist_ok=True)
        wrapper_path.unlink(missing_ok=True)
        wrapper_path.write_text(WRAPPER_CONTENT, encoding="utf-8")
        wrapper_path.chmod(0o755)
        return True, "native BUN_OPTIONS wrapper installed"
    else:
        content = p.get("content", "")
        if not content:
            return False, "no wrapper content in patch"
        if wrapper_path.exists():
            try:
                existing = wrapper_path.read_text()
                if JS_MARKER in existing or NATIVE_MARKER in existing:
                    return True, "no-op (already applied)"
            except Exception:
                pass
        if dry_run:
            return True, "would install cli.js wrapper"
        wrapper_path.parent.mkdir(parents=True, exist_ok=True)
        wrapper_path.unlink(missing_ok=True)
        wrapper_path.write_text(content, encoding="utf-8")
        wrapper_path.chmod(0o755)
        return True, "cli.js wrapper installed"


def _apply_binary_install(p: dict, kind: str, dry_run: bool = False) -> tuple[bool, str]:
    """binary_install: for npm layout, replace file in package. Native binary: N/A."""
    if kind == "bun_sea":
        return True, "no-op (N/A for native binary — seccomp is in Bun runtime, not VFS)"
    return False, "npm layout binary_install not implemented in vpcc"


def cmd_verify(args) -> int:
    target, kind = find_target()
    if not target:
        print(f"{R}claude-code not found{X}")
        return 2

    if kind == "bun_sea":
        # In-place verify: read whole binary, locate .bun section, check markers inside.
        data = bytearray(target.read_bytes())
        try:
            bun_off, bun_size = _find_bun_section(data)
        except Exception as e:
            print(f"{R}ELF parse failed: {e}{X}")
            return 2
        text = bytes(data[bun_off:bun_off + bun_size]).decode("utf-8", errors="surrogateescape")
    else:
        text = target.read_text(encoding="utf-8")

    required_missing = 0
    optional_missing = 0
    applied = 0
    for p in load_patches():
        if p.get("retired"):
            continue
        if p.get("type") != "js_replace":
            continue
        checkable = [s for s in p.get("patches", []) if s.get("applied_marker")]
        if not checkable:
            continue

        patch_applied = any(s["applied_marker"] in text for s in checkable)

        if patch_applied:
            applied += 1
            continue

        is_required = any(s.get("required", False) for s in checkable)
        if is_required:
            print(f"{R}{CROSS}{X} {p['id']}")
            required_missing += 1
        else:
            optional_missing += 1

    # Systemic failure: more patches unapplied than applied — batch verification probably failed.
    systemic = optional_missing > applied

    if required_missing:
        print(f"\n{R}{required_missing} required patch(es) not applied{X}")
        return 1

    if systemic:
        print(f"{R}{CROSS} {optional_missing} patch(es) not applied, {applied} applied{X}"
              f"  — run {B}vpcc patch{X} to apply")
        return 1

    if optional_missing:
        print(f"{G}{CHECK} {applied} patch(es) verified{X}  "
              f"{Y}({optional_missing} optional not applied — run 'vpcc patch' if unexpected){X}")
    else:
        print(f"{G}{CHECK} all patches verified{X}")
    return 0


def cmd_rollback(args) -> int:
    target, kind = find_target()
    if not target:
        print(f"{R}claude-code not found{X}")
        return 2
    baks = sorted(BACKUP_DIR.glob("claude.*.bak"))
    if not baks:
        print(f"{R}no backups in {BACKUP_DIR}{X}")
        return 1
    latest = baks[-1]
    mode = target.stat().st_mode & 0o7777
    target.unlink()
    shutil.copy2(latest, target)
    target.chmod(mode)
    print(f"{G}{CHECK} restored{X} {target} <- {latest.name}")
    return 0


def cmd_status(args) -> int:
    target, kind = find_target()
    patches = load_patches()
    print(f"{B}vpcc status{X}")
    print(f"  patches : {len(patches)}")
    if target:
        if kind == "bun_sea":
            if sys.platform == "darwin":
                label = "Bun SEA (Mach-O)"
            elif sys.platform == "win32":
                label = "Bun SEA (PE)"
            else:
                label = "Bun SEA (ELF)"
        else:
            label = "cli.js (JS)"
        print(f"  target  : {target}")
        print(f"  format  : {label}")
        print(f"  sha256  : {sha256_short(target)}")
        print(f"  size    : {target.stat().st_size // 1024 // 1024} MB")
    else:
        print(f"  target  : {R}NOT FOUND{X}")
    baks = sorted(BACKUP_DIR.glob("claude.*.bak"))
    print(f"  backups : {len(baks)}  ({BACKUP_DIR})")
    return 0


def cmd_list(args) -> int:
    for p in load_patches():
        print(f"  {p['id']:40s}  {p.get('description','')}")
    return 0


def cmd_self_update(args) -> int:
    """Pull latest patches/*.json from GitHub."""
    print(f"{B}vpcc self-update{X}  <- {_updater.REPO}@{_updater.BRANCH}")
    remote = _updater.remote_head_sha("patches")
    if not remote:
        print(f"{R}could not reach GitHub API{X}")
        return 2
    state = _updater.load_state()
    local = state.get("patches_commit")
    print(f"  local  : {local or '(unknown)'}")
    print(f"  remote : {remote}")
    if local == remote and not args.force:
        print(f"{G}{CHECK} already up to date{X}")
        return 0
    if args.dry_run:
        print(f"{Y}dry-run: would sync{X}")
        return 0
    changed, sha_or_err = _updater.sync_patches(PATCH_DIR, remote)
    if changed < 0:
        print(f"{R}{CROSS} sync failed — {sha_or_err}{X}")
        return 2
    print(f"{G}{CHECK} synced{X}  {changed} file(s) updated @ {sha_or_err[:7]}")
    if changed and not args.no_reapply:
        print(f"\n{B}re-applying patches{X}")
        class _P: dry_run = False
        return cmd_patch(_P())
    return 0


def cmd_autoheal(args) -> int:
    """Detect CC drift, re-verify, self-update + re-patch if broken."""
    return _updater.autoheal(
        find_target=find_target,
        sha256_short=sha256_short,
        load_patches=load_patches,
        cmd_verify_fn=cmd_verify,
        cmd_patch_fn=cmd_patch,
        cmd_rollback_fn=cmd_rollback,
        patch_dir=PATCH_DIR,
        force=args.force,
        quiet=args.quiet,
    )


def cmd_scan(args) -> int:
    """Sig-based offset discovery. Prints anchor offsets + regex hit status."""
    target, kind = find_target()
    if not target:
        print(f"{R}claude-code not found{X}")
        return 2

    needs_text = bool(getattr(args, "auto_heal", False) or args.export_patch)
    rows = None
    text = None
    if not needs_text:
        rows = _scanner.load_cached_rows(target, PATCH_DIR)
    if rows is None:
        try:
            text = _scanner.load_text_from_target(target, kind)
        except Exception as e:
            print(f"{R}extract failed: {e}{X}")
            return 2
        patches = _scanner.load_patches_from_dir(PATCH_DIR)
        sc = _scanner.SigScanner(text)
        rows = sc.scan_patches(patches)
        _scanner.save_cached_rows(target, PATCH_DIR, rows)
    else:
        sc = None  # cached path — no live scanner

    print(f"{B}vpcc scan — {len(rows)} js_replace patches{X}")
    print(f"  target : {target}")
    if text is not None:
        print(f"  format : {'cli.js' if kind == 'js' else '.bun section (' + str(len(text)) + ' bytes)'}")
    else:
        print(f"  format : {'cli.js' if kind == 'js' else 'Bun SEA'} (cached)")
    print(f"  sha    : {sha256_short(target)}")
    print()
    print(_scanner.format_scan_report(rows, verbose=args.verbose))

    if getattr(args, "auto_heal", False):
        res = _scanner.auto_heal_drift(text, PATCH_DIR, verbose=args.verbose)
        print(f"\n{B}auto-heal:{X}  {G}{res['healed']} healed{X}  "
              f"{res['skipped']} skipped  {R}{res['failed']} failed{X}")

    if args.export_patch:
        target_id = args.export_patch
        match = next((r for r in rows if r["id"] == target_id), None)
        if not match:
            print(f"\n{R}no patch id '{target_id}'{X}")
            return 1
        anchors = match["anchors"]
        if not anchors:
            print(f"\n{R}patch has no anchor_strings — cannot regenerate{X}")
            return 1
        regex = sc.derive_regex(anchors[0])
        print(f"\n{B}regenerated regex for {target_id}:{X}")
        print(f"  {regex}")

    return 1 if any(r["status"] == "drift" for r in rows) else 0


def cmd_doctor(args) -> int:
    """Full health report: sha, patches applied, sig drift, backup count, upstream."""
    target, kind = find_target()
    patches = load_patches()
    n_retired = sum(
        1 for f in sorted(PATCH_DIR.glob("*.json"))
        if _is_retired_json(f)
    )
    print(f"{B}vpcc doctor{X}")
    print(f"  vpcc ver   : {__import__('vpcc').__version__ if hasattr(__import__('vpcc'), '__version__') else '2.1.114'}")
    print(f"  patches    : {len(patches)}")
    if not target:
        print(f"  target     : {R}NOT FOUND{X}")
        return 2
    if kind == "bun_sea":
        if sys.platform == "darwin":
            label = "Bun SEA (Mach-O)"
        elif sys.platform == "win32":
            label = "Bun SEA (PE)"
        else:
            label = "Bun SEA (ELF)"
    else:
        label = "cli.js (JS)"
    print(f"  target     : {target}")
    print(f"  format     : {label}")
    print(f"  sha256     : {sha256_short(target)}")
    print(f"  size       : {target.stat().st_size // 1024 // 1024} MB")

    # sig drift
    try:
        rows = _scanner.load_cached_rows(target, PATCH_DIR)
        if rows is None:
            text = _scanner.load_text_from_target(target, kind)
            rows = _scanner.SigScanner(text).scan_patches(_scanner.load_patches_from_dir(PATCH_DIR))
            _scanner.save_cached_rows(target, PATCH_DIR, rows)
        drift = [r["id"] for r in rows if r["status"] == "drift"]
        nometa = [r["id"] for r in rows if r["status"] == "unclassified"]
        if drift:
            print(f"  {R}sig drift  : {len(drift)} patches {DOT} {', '.join(drift[:3])}{'...' if len(drift) > 3 else ''}{X}")
        else:
            print(f"  {G}sig drift  : 0 (all anchors locatable){X}")
        if nometa:
            print(f"  {Y}sig nometa : {len(nometa)} patches (pre-v2.1.114, no anchor_strings yet){X}")
    except Exception as e:
        print(f"  {R}sig scan   : failed — {e}{X}")
    print(f"  retired    : {n_retired} patches")

    # applied markers
    try:
        rc_v = cmd_verify(type("A", (), {})())
    except SystemExit:
        rc_v = 1
    print(f"  applied    : {'all' if rc_v == 0 else 'partial/none'}")

    # backups
    baks = sorted(BACKUP_DIR.glob("claude.*.bak"))
    print(f"  backups    : {len(baks)} in {BACKUP_DIR}")

    # upstream
    try:
        info = _updater.upstream_status(PATCH_DIR)
        if info["drift"]:
            print(f"  {Y}upstream   : behind — run 'vpcc self-update'{X}")
        elif info["remote_commit"]:
            print(f"  {G}upstream   : current{X}")
        else:
            print(f"  upstream   : unreachable")
    except Exception:
        print(f"  upstream   : error")

    return 0



def cmd_guard(args) -> int:
    """Fast SHA-based guard: if CC binary changed since last patch, run autopilot.

    Designed to be called from wrapper scripts before every `claude` invocation.
    <100ms when no update needed (single SHA comparison against stamp file).
    On change: runs full autopilot (scan → heal → patch → commit → push → issue).
    """
    target, kind = find_target()
    if not target:
        return 0  # no CC found, nothing to guard

    stamp = Path.home() / ".vpcc" / "last_patched_sha"
    cur_sha = sha256_short(target)

    if stamp.is_file():
        try:
            if stamp.read_text().strip() == cur_sha:
                return 0  # binary unchanged, nothing to do
        except OSError:
            pass

    # Binary changed — run full autopilot pipeline
    print(f"{B}vpcc guard — CC binary changed ({cur_sha}), running autopilot...{X}")
    backup(target, kind)
    rc = cmd_autopilot(type("A", (), {})())

    # Stamp the new SHA regardless of outcome (avoid infinite retry loops)
    try:
        stamp.parent.mkdir(parents=True, exist_ok=True)
        stamp.write_text(sha256_short(target))
    except OSError:
        pass

    return rc


def cmd_install_guard(args) -> int:
    """Install platform-specific background guard (auto-patch on CC update).

    Windows: Task Scheduler task (every 6h + on logon)
    macOS: launchd plist (every 6h)
    Linux: systemd --user timer (every 6h)
    """
    vpcc_bin = shutil.which("vpcc") or str(Path(sys.executable).parent / "vpcc")

    if sys.platform == "win32":
        # Windows Task Scheduler
        task_name = "vpcc-autoheal"
        # Delete existing task first (idempotent)
        subprocess.run(["schtasks", "/Delete", "/TN", task_name, "/F"],
                       capture_output=True, timeout=10)
        # Create: run every 6 hours + on logon
        rc = subprocess.run([
            "schtasks", "/Create", "/TN", task_name,
            "/TR", f'"{vpcc_bin}" guard',
            "/SC", "HOURLY", "/MO", "6",
            "/RL", "LIMITED",
            "/F",
        ], capture_output=True, text=True, timeout=15)
        if rc.returncode != 0:
            print(f"{R}Task Scheduler failed: {rc.stderr.strip()}{X}")
            return 1
        # Add logon trigger
        subprocess.run([
            "schtasks", "/Change", "/TN", task_name,
            "/ENABLE",
        ], capture_output=True, timeout=10)
        print(f"{G}{CHECK} Windows Task Scheduler: {task_name} (every 6h + logon){X}")

    elif sys.platform == "darwin":
        # macOS launchd
        plist_dir = Path.home() / "Library" / "LaunchAgents"
        plist_dir.mkdir(parents=True, exist_ok=True)
        plist_path = plist_dir / "cc.voidchecksum.vpcc-guard.plist"
        plist_content = f"""<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>cc.voidchecksum.vpcc-guard</string>
    <key>ProgramArguments</key>
    <array>
        <string>{vpcc_bin}</string>
        <string>guard</string>
    </array>
    <key>StartInterval</key>
    <integer>21600</integer>
    <key>RunAtLoad</key>
    <true/>
    <key>StandardOutPath</key>
    <string>{Path.home()}/.vpcc/guard.log</string>
    <key>StandardErrorPath</key>
    <string>{Path.home()}/.vpcc/guard.log</string>
    <key>Nice</key>
    <integer>10</integer>
</dict>
</plist>
"""
        plist_path.write_text(plist_content)
        subprocess.run(["launchctl", "unload", str(plist_path)], capture_output=True, timeout=10)
        subprocess.run(["launchctl", "load", str(plist_path)], capture_output=True, timeout=10)
        print(f"{G}{CHECK} macOS launchd: {plist_path.name} (every 6h + login){X}")

    else:
        # Linux systemd --user
        unit_dir = Path(os.environ.get("XDG_CONFIG_HOME", str(Path.home() / ".config"))) / "systemd" / "user"
        unit_dir.mkdir(parents=True, exist_ok=True)

        svc = unit_dir / "vpcc-guard.service"
        svc.write_text(f"""[Unit]
Description=vpcc guard — auto-patch Claude Code on update
Documentation=https://github.com/VoidChecksum/void-patcher-cc
Wants=network-online.target
After=network-online.target

[Service]
Type=oneshot
ExecStart={vpcc_bin} guard
Nice=10

[Install]
WantedBy=default.target
""")
        tmr = unit_dir / "vpcc-guard.timer"
        tmr.write_text("""[Unit]
Description=Run vpcc guard periodically
Documentation=https://github.com/VoidChecksum/void-patcher-cc

[Timer]
OnBootSec=2min
OnUnitActiveSec=6h
RandomizedDelaySec=15min
Persistent=true
Unit=vpcc-guard.service

[Install]
WantedBy=timers.target
""")
        subprocess.run(["systemctl", "--user", "daemon-reload"], capture_output=True, timeout=10)
        subprocess.run(["systemctl", "--user", "enable", "--now", "vpcc-guard.timer"],
                       capture_output=True, timeout=10)
        print(f"{G}{CHECK} systemd --user: vpcc-guard.timer (every 6h + boot){X}")

    print(f"  vpcc guard runs automatically — Claude Code updates are patched within minutes")
    return 0


def cmd_uninstall_guard(args) -> int:
    """Remove platform-specific background guard."""
    if sys.platform == "win32":
        r = subprocess.run(["schtasks", "/Delete", "/TN", "vpcc-autoheal", "/F"],
                           capture_output=True, text=True, timeout=10)
        print(f"{G}{CHECK} removed Windows scheduled task{X}" if r.returncode == 0 else f"{Y}no task found{X}")
    elif sys.platform == "darwin":
        plist = Path.home() / "Library" / "LaunchAgents" / "cc.voidchecksum.vpcc-guard.plist"
        if plist.exists():
            subprocess.run(["launchctl", "unload", str(plist)], capture_output=True, timeout=10)
            plist.unlink()
            print(f"{G}{CHECK} removed macOS launchd agent{X}")
        else:
            print(f"{Y}no launchd agent found{X}")
    else:
        unit_dir = Path(os.environ.get("XDG_CONFIG_HOME", str(Path.home() / ".config"))) / "systemd" / "user"
        subprocess.run(["systemctl", "--user", "disable", "--now", "vpcc-guard.timer"],
                       capture_output=True, timeout=10)
        for name in ("vpcc-guard.service", "vpcc-guard.timer"):
            f = unit_dir / name
            if f.exists():
                f.unlink()
        subprocess.run(["systemctl", "--user", "daemon-reload"], capture_output=True, timeout=10)
        print(f"{G}{CHECK} removed systemd timer{X}")

    # Remove stamp file
    stamp = Path.home() / ".vpcc" / "last_patched_sha"
    stamp.unlink(missing_ok=True)
    return 0

def _detect_install_method() -> str:
    """Detect how vpcc was installed. Returns one of: pipx, pip, uv, git, unknown."""
    vpcc_path = Path(__file__).resolve()
    vpcc_str = str(vpcc_path).lower().replace("\\", "/")

    if "/pipx/" in vpcc_str:
        return "pipx"
    if "/uv/" in vpcc_str or "/.local/share/uv/" in vpcc_str:
        return "uv"

    # Check if we're in a git repo (developer install / editable)
    git_dir = vpcc_path.parent
    for _ in range(5):
        if (git_dir / ".git").exists():
            return "git"
        git_dir = git_dir.parent

    # pip (any venv or user install)
    return "pip"


def cmd_update(args) -> int:
    """OMP-style full self-update: upgrade vpcc itself + patches + re-patch binary.

    Detects install method (pipx/pip/uv/git) and runs the right upgrade command,
    then syncs patches from GitHub and re-applies them to the CC binary.

    This is the single command users should run after a CC update or periodically.
    """
    from vpcc import __version__

    print(f"{B}vpcc update{X}")
    print(f"  current : v{__version__}")
    method = _detect_install_method()
    print(f"  install : {method}")

    REPO_URL = f"git+https://github.com/{_updater.REPO}"

    # ── Step 1: Check if a new version is available ───────────────────────
    print(f"\n  {B}[1/4] checking for updates...{X}")
    remote_ver = None
    try:
        raw = _updater._req(
            f"https://raw.githubusercontent.com/{_updater.REPO}/{_updater.BRANCH}/vpcc/__init__.py",
            accept="text/plain", timeout=10,
        ).decode("utf-8", errors="replace")
        for line in raw.splitlines():
            if line.startswith("__version__"):
                remote_ver = line.split('"')[1]
                break
    except Exception:
        pass

    if remote_ver:
        print(f"  remote  : v{remote_ver}")
        if remote_ver == __version__:
            print(f"  {G}{CHECK} vpcc is already at the latest version{X}")
            needs_tool_update = False
        else:
            needs_tool_update = True
    else:
        print(f"  {Y}{WARN_ICON} could not check remote version — updating anyway{X}")
        needs_tool_update = True

    # ── Step 2: Update vpcc itself ────────────────────────────────────────
    if needs_tool_update:
        print(f"\n  {B}[2/4] updating vpcc ({method})...{X}")
        rc = _run_tool_update(method, REPO_URL)
        if rc == 0:
            print(f"  {G}{CHECK} vpcc updated{X}")
        else:
            print(f"  {Y}{WARN_ICON} tool update returned rc={rc} — continuing with patch sync{X}")
    else:
        print(f"\n  {B}[2/4] skipping tool update (already current){X}")

    # ── Step 3: Sync patches from GitHub ──────────────────────────────────
    print(f"\n  {B}[3/4] syncing patches...{X}")
    remote_sha = _updater.remote_head_sha("patches")
    if remote_sha:
        state = _updater.load_state()
        local_sha = state.get("patches_commit")
        if local_sha == remote_sha:
            print(f"  {G}{CHECK} patches already at latest ({remote_sha[:7]}){X}")
        else:
            changed, sha_or_err = _updater.sync_patches(PATCH_DIR, remote_sha)
            if changed >= 0:
                print(f"  {G}{CHECK} {changed} patch file(s) synced @ {sha_or_err[:7]}{X}")
            else:
                print(f"  {R}{CROSS} patch sync failed: {sha_or_err}{X}")
    else:
        print(f"  {Y}{WARN_ICON} could not reach GitHub — using local patches{X}")

    # ── Step 4: Re-apply patches to CC binary ─────────────────────────────
    print(f"\n  {B}[4/4] patching Claude Code binary...{X}")
    target, kind = find_target()
    if target:
        class _P: dry_run = False
        rc_patch = cmd_patch(_P())
        if rc_patch == 0:
            # Stamp guard SHA
            try:
                stamp = Path.home() / ".vpcc" / "last_patched_sha"
                stamp.parent.mkdir(parents=True, exist_ok=True)
                stamp.write_text(sha256_short(target))
            except OSError:
                pass
    else:
        print(f"  {Y}{WARN_ICON} Claude Code not found — skipping binary patch{X}")
        rc_patch = 0

    # ── Summary ───────────────────────────────────────────────────────────
    print(f"\n{B}{'─' * 40}{X}")
    if rc_patch == 0:
        print(f"  {G}{CHECK} vpcc update complete{X}")
    else:
        print(f"  {Y}{WARN_ICON} update complete with warnings (rc={rc_patch}){X}")

    # Show new version if we updated
    if needs_tool_update:
        try:
            r = subprocess.run(
                [sys.executable, "-c", "from vpcc import __version__; print(__version__)"],
                capture_output=True, text=True, timeout=5)
            new_ver = r.stdout.strip()
            if new_ver and new_ver != __version__:
                print(f"  {G}v{__version__} {ARROW} v{new_ver}{X}")
        except Exception:
            pass

    return rc_patch


def _run_tool_update(method: str, repo_url: str) -> int:
    """Run the appropriate package manager command to upgrade vpcc."""
    cmds: dict[str, list[list[str]]] = {
        "pipx": [
            ["pipx", "install", "--force", repo_url],
        ],
        "uv": [
            ["uv", "tool", "install", "--force", repo_url],
        ],
        "pip": [
            [sys.executable, "-m", "pip", "install", "--upgrade", "--force-reinstall", repo_url],
        ],
        "git": [
            # For git installs: pull latest + reinstall in editable mode
            ["git", "pull", "--ff-only"],
        ],
    }

    for cmd in cmds.get(method, cmds["pip"]):
        try:
            # For git pull, run in the repo directory
            cwd = str(ROOT) if method == "git" and cmd[0] == "git" else None
            r = subprocess.run(cmd, capture_output=True, text=True, timeout=120, cwd=cwd)
            if r.returncode != 0:
                err = (r.stderr or r.stdout or "").strip().splitlines()
                if err:
                    print(f"  {Y}{err[-1][:100]}{X}")
                return r.returncode
        except FileNotFoundError:
            print(f"  {Y}{WARN_ICON} '{cmd[0]}' not found — trying pip fallback{X}")
            # Fallback to pip
            try:
                r = subprocess.run(
                    [sys.executable, "-m", "pip", "install", "--upgrade", "--force-reinstall", repo_url],
                    capture_output=True, text=True, timeout=120)
                return r.returncode
            except Exception:
                return 1
        except Exception as e:
            print(f"  {R}{e}{X}")
            return 1
    return 0

def cmd_autopilot(args) -> int:
    """Full automatic pipeline: detect drift → auto-heal → patch → commit → push → issue.

    Runs the entire vpcc lifecycle unattended:
    1. Scan binary for signature drift
    2. Auto-heal any drifted patches from anchor context windows
    3. Re-apply all patches to the binary
    4. If this is a git repo, commit healed patches and push to origin
    5. If any patches couldn't be healed, open a GitHub issue

    Exit codes: 0=all good, 1=partial (some drift unfixable), 2=fatal error
    """
    target, kind = find_target()
    if not target:
        print(f"{R}claude-code not found{X}")
        return 2

    ts = datetime.now().strftime("%H:%M:%S")
    cur_sha = sha256_short(target)
    print(f"{B}vpcc autopilot — {ts}{X}")
    print(f"  target : {target}")
    print(f"  sha    : {cur_sha}")

    # ── 1. Scan for drift ─────────────────────────────────────────────────
    print(f"\n  {B}[1/5] scanning...{X}")
    try:
        text = _scanner.load_text_from_target(target, kind)
    except Exception as e:
        print(f"  {R}extract failed: {e}{X}")
        return 2
    patches = _scanner.load_patches_from_dir(PATCH_DIR)
    sc = _scanner.SigScanner(text)
    rows = sc.scan_patches(patches)

    n_ok = sum(1 for r in rows if r["status"] == "ok")
    n_applied = sum(1 for r in rows if r["status"] == "applied")
    n_drift = sum(1 for r in rows if r["status"] == "drift")
    n_retired = sum(1 for r in rows if r["status"] == "retired")
    print(f"  {G}{n_ok} ok{X}  {n_applied} applied  {R}{n_drift} drift{X}  {n_retired} retired")

    if n_drift == 0:
        print(f"  {G}no drift — skipping heal{X}")
    else:
        # ── 2. Auto-heal drifted patches ──────────────────────────────────
        print(f"\n  {B}[2/5] auto-healing {n_drift} drifted patches...{X}")
        heal_result = _scanner.auto_heal_drift(text, PATCH_DIR, verbose=True)
        healed = heal_result["healed"]
        failed = heal_result["failed"]
        print(f"  {G}{healed} healed{X}  {R}{failed} failed{X}")

    # ── 3. Apply patches ──────────────────────────────────────────────────
    print(f"\n  {B}[3/5] patching binary...{X}")
    class _P: dry_run = False
    rc_patch = cmd_patch(_P())

    # ── 4. Git commit + push (if in a git repo with changes) ─────────────
    print(f"\n  {B}[4/5] checking git...{X}")
    git_pushed = False
    repo_root = _find_git_root()
    if repo_root:
        rc_diff = subprocess.run(
            ["git", "diff", "--quiet", "--", "patches/"],
            cwd=str(repo_root), capture_output=True, timeout=10,
        )
        has_changes = rc_diff.returncode != 0
        # Also check for untracked new patches
        rc_untracked = subprocess.run(
            ["git", "ls-files", "--others", "--exclude-standard", "patches/"],
            cwd=str(repo_root), capture_output=True, text=True, timeout=10,
        )
        has_new = bool(rc_untracked.stdout.strip())

        if has_changes or has_new:
            # Get CC version for commit message
            cc_ver = _detect_cc_version(target)
            msg = (f"fix(patches): auto-heal for Claude Code {cc_ver}\n\n"
                   f"Binary SHA: {cur_sha}\n"
                   f"Healed: {heal_result['healed'] if n_drift else 0} patches\n"
                   f"Scan: {n_ok} ok, {n_applied} applied, {n_drift} drift\n\n"
                   f"[autopilot]")
            subprocess.run(["git", "add", "patches/"], cwd=str(repo_root),
                           capture_output=True, timeout=10)
            rc_commit = subprocess.run(
                ["git", "commit", "-m", msg],
                cwd=str(repo_root), capture_output=True, text=True, timeout=15,
            )
            if rc_commit.returncode == 0:
                print(f"  {G}{CHECK} committed:{X} {rc_commit.stdout.strip().splitlines()[0]}")
                rc_push = subprocess.run(
                    ["git", "push", "origin", "HEAD"],
                    cwd=str(repo_root), capture_output=True, text=True, timeout=60,
                )
                if rc_push.returncode == 0:
                    print(f"  {G}{CHECK} pushed to origin{X}")
                    git_pushed = True
                else:
                    print(f"  {Y}{WARN_ICON} push failed: {rc_push.stderr.strip()[:80]}{X}")
            else:
                print(f"  {Y}nothing to commit{X}")
        else:
            print(f"  {G}no patch changes to commit{X}")
    else:
        print(f"  {Y}not a git repo — skip commit/push{X}")

    # ── 5. Create GitHub issue for unfixable drift ────────────────────────
    unfixed = []
    if n_drift > 0:
        # Re-scan after healing to see what's still broken
        rows2 = _scanner.SigScanner(text).scan_patches(
            _scanner.load_patches_from_dir(PATCH_DIR))
        unfixed = [r for r in rows2 if r["status"] == "drift"]

    print(f"\n  {B}[5/5] issue check...{X}")
    if unfixed:
        cc_ver = _detect_cc_version(target)
        issue_title = f"Signature drift: {len(unfixed)} patches broken on CC {cc_ver}"
        issue_body = (
            f"## Autopilot drift report\n\n"
            f"- **Claude Code version**: `{cc_ver}`\n"
            f"- **Binary SHA**: `{cur_sha}`\n"
            f"- **Unfixed patches**: {len(unfixed)}\n\n"
            f"| Patch | Anchors |\n|---|---|\n"
        )
        for u in unfixed:
            anchors = ", ".join(f"`{a[:40]}`" for a in (u.get("anchors") or [])[:2])
            issue_body += f"| `{u['id']}` | {anchors or 'none'} |\n"
        issue_body += (
            f"\n### Action needed\n"
            f"RE the binary at `{target}` and update `search_regex` + `anchor_strings` "
            f"in the affected `patches/*.json` files.\n\n"
            f"*Generated by `vpcc autopilot`*"
        )
        try:
            rc_issue = subprocess.run(
                ["gh", "issue", "create",
                 "--title", issue_title,
                 "--body", issue_body,
                 "--label", "signature-drift"],
                cwd=str(repo_root) if repo_root else None,
                capture_output=True, text=True, timeout=30,
            )
            if rc_issue.returncode == 0:
                print(f"  {G}{CHECK} issue created:{X} {rc_issue.stdout.strip()}")
            else:
                print(f"  {Y}{WARN_ICON} gh issue failed (install 'gh' CLI for auto-issues){X}")
        except FileNotFoundError:
            print(f"  {Y}{WARN_ICON} 'gh' CLI not found — install for auto-issues{X}")
        except Exception:
            print(f"  {Y}{WARN_ICON} issue creation failed{X}")
    else:
        print(f"  {G}no unfixed drift — no issue needed{X}")

    # ── Summary ───────────────────────────────────────────────────────────
    print(f"\n{B}{'─' * 50}{X}")
    status_icon = CHECK if not unfixed and rc_patch == 0 else WARN_ICON if unfixed else CROSS
    print(f"  {G if not unfixed else Y}{status_icon} autopilot complete{X}")
    print(f"  patches : {rc_patch == 0 and 'applied' or 'failed'}")
    if n_drift:
        print(f"  healed  : {heal_result['healed']}/{n_drift}")
    if git_pushed:
        print(f"  git     : committed + pushed")
    if unfixed:
        print(f"  {R}unfixed : {len(unfixed)} patches need manual RE{X}")
    return 1 if unfixed else (2 if rc_patch != 0 else 0)


def _find_git_root() -> Path | None:
    """Find the git repo root containing PATCH_DIR, if any."""
    try:
        r = subprocess.run(
            ["git", "rev-parse", "--show-toplevel"],
            cwd=str(PATCH_DIR), capture_output=True, text=True, timeout=5,
        )
        if r.returncode == 0 and r.stdout.strip():
            return Path(r.stdout.strip())
    except Exception:
        pass
    return None


def _detect_cc_version(target: Path) -> str:
    """Run the binary with --version and extract the version string."""
    try:
        r = subprocess.run([str(target), "--version"],
                           capture_output=True, text=True, timeout=15)
        for line in (r.stdout + r.stderr).splitlines():
            line = line.strip()
            if line and line[0].isdigit():
                return line.split()[0]
    except Exception:
        pass
    return target.name


def cmd_dashboard(args) -> int:
    """Real-time status dashboard with auto-refresh.

    Shows: CC version, SHA, patch status, guard status, last heal time.
    Refreshes every --interval seconds. Ctrl-C to exit.
    """
    import time

    interval = getattr(args, "interval", 5)
    CLEAR = "\033[2J\033[H"  # ANSI clear screen + home
    DIM = "\033[90m"

    try:
        while True:
            target, kind = find_target()
            ts = datetime.now().strftime("%Y-%m-%d %H:%M:%S")

            lines = [
                f"{CLEAR}{B}vpcc dashboard{X}  {DIM}{ts}{X}",
                f"{DIM}{'─' * 60}{X}",
            ]

            if not target:
                lines.append(f"  {R}claude-code not found{X}")
            else:
                cur_sha = sha256_short(target)
                cc_ver = _detect_cc_version(target)
                size_mb = target.stat().st_size // (1024 * 1024)

                lines.append(f"  {B}target{X}   : {target}")
                lines.append(f"  {B}version{X}  : {cc_ver}")
                lines.append(f"  {B}sha256{X}   : {cur_sha}")
                lines.append(f"  {B}size{X}     : {size_mb} MB")
                lines.append(f"  {B}format{X}   : {'Bun SEA' if kind == 'bun_sea' else 'cli.js'}")

                # Guard stamp
                stamp = Path.home() / ".vpcc" / "last_patched_sha"
                if stamp.is_file():
                    stamp_sha = stamp.read_text().strip()
                    stamp_age = time.time() - stamp.stat().st_mtime
                    age_str = _human_age(stamp_age)
                    if stamp_sha == cur_sha:
                        lines.append(f"  {B}guard{X}    : {G}{CHECK} patched ({age_str} ago){X}")
                    else:
                        lines.append(f"  {B}guard{X}    : {R}{CROSS} stale — binary changed since last patch{X}")
                else:
                    lines.append(f"  {B}guard{X}    : {Y}{WARN_ICON} no stamp — run 'vpcc patch'{X}")

                # Scan cache
                cached = _scanner.load_cached_rows(target, PATCH_DIR)
                if cached:
                    n_ok = sum(1 for r in cached if r["status"] == "ok")
                    n_applied = sum(1 for r in cached if r["status"] == "applied")
                    n_drift = sum(1 for r in cached if r["status"] == "drift")
                    n_retired = sum(1 for r in cached if r["status"] == "retired")
                    n_total = n_ok + n_applied + n_drift
                    lines.append("")
                    lines.append(f"  {B}patches{X}  : {n_total} scanned")
                    lines.append(f"    {G}{CHECK} {n_ok} ok{X}  "
                                 f"{G}{CHECK} {n_applied} applied{X}  "
                                 f"{R + CROSS + ' ' + str(n_drift) + ' drift' + X if n_drift else DIM + '0 drift' + X}  "
                                 f"{DIM}{n_retired} retired{X}")
                    if n_drift:
                        drift_ids = [r["id"] for r in cached if r["status"] == "drift"]
                        for did in drift_ids[:5]:
                            lines.append(f"    {R}{ARROW} {did}{X}")
                        if len(drift_ids) > 5:
                            lines.append(f"    {DIM}... and {len(drift_ids) - 5} more{X}")
                else:
                    lines.append(f"\n  {B}patches{X}  : {DIM}(run 'vpcc scan' to populate cache){X}")

                # Backups
                baks = sorted(BACKUP_DIR.glob("claude.*.bak"))
                lines.append(f"\n  {B}backups{X}  : {len(baks)} in {BACKUP_DIR}")

                # Guard scheduler
                guard_status = _detect_guard_scheduler()
                lines.append(f"  {B}scheduler{X}: {guard_status}")

                # Upstream
                try:
                    info = _updater.upstream_status(PATCH_DIR)
                    if info["drift"]:
                        lines.append(f"  {B}upstream{X} : {Y}behind — run 'vpcc self-update'{X}")
                    elif info["remote_commit"]:
                        lines.append(f"  {B}upstream{X} : {G}current{X}")
                    else:
                        lines.append(f"  {B}upstream{X} : {DIM}unreachable{X}")
                except Exception:
                    lines.append(f"  {B}upstream{X} : {DIM}error{X}")

            lines.append(f"\n{DIM}refreshing every {interval}s — Ctrl-C to exit{X}")
            print("\n".join(lines), flush=True)
            time.sleep(interval)

    except KeyboardInterrupt:
        print(f"\n{B}dashboard stopped{X}")
        return 0


def _human_age(seconds: float) -> str:
    """Convert seconds to human-readable age string."""
    if seconds < 60:
        return f"{int(seconds)}s"
    if seconds < 3600:
        return f"{int(seconds // 60)}m"
    if seconds < 86400:
        return f"{int(seconds // 3600)}h"
    return f"{int(seconds // 86400)}d"


def _detect_guard_scheduler() -> str:
    """Detect if a background guard scheduler is installed."""
    if sys.platform == "win32":
        try:
            r = subprocess.run(
                ["schtasks", "/Query", "/TN", "vpcc-autoheal", "/FO", "LIST"],
                capture_output=True, text=True, timeout=5,
            )
            if r.returncode == 0:
                return f"{G}Windows Task Scheduler active{X}"
        except Exception:
            pass
        return f"{Y}not installed — run 'vpcc install-guard'{X}"
    elif sys.platform == "darwin":
        plist = Path.home() / "Library" / "LaunchAgents" / "cc.voidchecksum.vpcc-guard.plist"
        if plist.exists():
            return f"{G}macOS launchd active{X}"
        return f"{Y}not installed — run 'vpcc install-guard'{X}"
    else:
        try:
            r = subprocess.run(
                ["systemctl", "--user", "is-active", "vpcc-guard.timer"],
                capture_output=True, text=True, timeout=5,
            )
            if r.stdout.strip() == "active":
                return f"{G}systemd timer active{X}"
        except Exception:
            pass
        return f"{Y}not installed — run 'vpcc install-guard'{X}"

def cmd_watch(args) -> int:
    """Daemon: poll binary mtime+sha; on change → full autopilot pipeline.

    Replaces simple autoheal with the full autopilot flow:
    scan → heal → patch → git commit/push → GitHub issue for unfixable drift.
    """
    import time
    target, kind = find_target()
    if not target:
        print(f"{R}claude-code not found{X}")
        return 2

    DIM = "\033[90m"
    print(f"{B}vpcc watch — polling every {args.interval}s{X}")
    print(f"  target : {target}")

    last_sha = sha256_short(target)
    last_mtime = target.stat().st_mtime
    cc_ver = _detect_cc_version(target)
    print(f"  version: {cc_ver}")
    print(f"  sha    : {last_sha}")
    print(f"  {DIM}watching for changes... (Ctrl-C to stop){X}")

    try:
        while True:
            time.sleep(args.interval)
            try:
                target, kind = find_target()
                if not target:
                    continue
                m = target.stat().st_mtime
                if m == last_mtime:
                    continue
                cur_sha = sha256_short(target)
                if cur_sha == last_sha:
                    last_mtime = m
                    continue

                ts = datetime.now().strftime("%H:%M:%S")
                new_ver = _detect_cc_version(target)
                print(f"\n{Y}[{ts}] CC updated: {cc_ver} ({last_sha}) {ARROW} {new_ver} ({cur_sha}){X}")
                backup(target, kind)

                # Run full autopilot (scan → heal → patch → commit → push → issue)
                rc = cmd_autopilot(type("A", (), {})())
                print(f"  autopilot rc={rc}")

                # Update stamp
                try:
                    stamp = Path.home() / ".vpcc" / "last_patched_sha"
                    stamp.parent.mkdir(parents=True, exist_ok=True)
                    stamp.write_text(sha256_short(target))
                except OSError:
                    pass

                last_sha = sha256_short(target)
                last_mtime = target.stat().st_mtime
                cc_ver = new_ver
                print(f"  {DIM}watching for changes...{X}")

            except Exception as e:
                print(f"{R}  watch error: {e}{X}")
    except KeyboardInterrupt:
        print(f"\n{B}watch stopped{X}")
        return 0


def _deep_merge(dst: dict, src: dict) -> dict:
    """Recursively merge src into dst. src wins on conflict. Arrays concatenated uniquely."""
    for k, v in src.items():
        if k in dst and isinstance(dst[k], dict) and isinstance(v, dict):
            _deep_merge(dst[k], v)
        elif k in dst and isinstance(dst[k], list) and isinstance(v, list):
            merged = list(dst[k])
            for item in v:
                if item not in merged:
                    merged.append(item)
            dst[k] = merged
        else:
            dst[k] = v
    return dst


_VPCC_CLAUDE_MD_START = "<!-- vpcc:authorization:start -->"
_VPCC_CLAUDE_MD_END   = "<!-- vpcc:authorization:end -->"


def cmd_upgrade(args) -> int:
    """All-in-one upgrade: self-update patches → autoheal → verify → warm scan cache.

    Idempotent and safe: each step short-circuits when nothing to do.
    Use this after a Claude Code auto-update.
    """
    print(f"{B}vpcc upgrade — full pipeline{X}")
    print(f"  step 1/4: self-update patches")
    rc_su = cmd_self_update(type("A", (), {"force": False})())
    if rc_su not in (0, 1):
        print(f"  {Y}self-update returned rc={rc_su}, continuing{X}")

    print(f"\n  step 2/4: autoheal (force)")
    rc_ah = cmd_autoheal(type("A", (), {"force": True, "quiet": False})())
    if rc_ah == 3:
        print(f"  {R}autoheal failed — aborting upgrade{X}")
        return 3

    print(f"\n  step 3/4: verify")
    rc_v = cmd_verify(type("A", (), {})())
    if rc_v != 0:
        print(f"  {R}verify failed — manual intervention required{X}")
        return 3

    print(f"\n  step 4/4: warm scan cache")
    target, kind = find_target()
    if target:
        try:
            text = _scanner.load_text_from_target(target, kind)
            patches = _scanner.load_patches_from_dir(PATCH_DIR)
            rows = _scanner.SigScanner(text).scan_patches(patches)
            _scanner.save_cached_rows(target, PATCH_DIR, rows)
            drift = sum(1 for r in rows if r["status"] == "drift")
            print(f"  cache warmed: {len(rows)} patches, {drift} drift")
        except Exception as e:
            print(f"  {Y}cache warm skipped: {e}{X}")

    print(f"\n{G}{CHECK} upgrade complete{X}")
    return 0


def cmd_bench(args) -> int:
    """Microbench: sha256, text load, scan cold, scan cached."""
    import time
    target, kind = find_target()
    if not target:
        print(f"{R}claude-code not found{X}")
        return 2

    print(f"{B}vpcc bench — {target.name} ({target.stat().st_size // 1024 // 1024} MB){X}")

    t0 = time.perf_counter()
    _ = sha256_short(target)
    print(f"  sha256             : {(time.perf_counter()-t0)*1000:6.1f} ms")

    t0 = time.perf_counter()
    text = _scanner.load_text_from_target(target, kind)
    t_load = (time.perf_counter()-t0)*1000
    print(f"  text load+decode   : {t_load:6.1f} ms  ({len(text)//1024//1024} MB)")

    patches = _scanner.load_patches_from_dir(PATCH_DIR)
    t0 = time.perf_counter()
    rows = _scanner.SigScanner(text).scan_patches(patches)
    t_scan = (time.perf_counter()-t0)*1000
    print(f"  scan_patches       : {t_scan:6.1f} ms  ({len(rows)} patches)")

    _scanner.save_cached_rows(target, PATCH_DIR, rows)
    t0 = time.perf_counter()
    _ = _scanner.load_cached_rows(target, PATCH_DIR)
    t_cache = (time.perf_counter()-t0)*1000
    print(f"  cache hit          : {t_cache:6.1f} ms  (key: sha256 + patch mtime)")

    print(f"\n  cold total         : {t_load + t_scan:6.1f} ms")
    print(f"  warm total         : {t_cache:6.1f} ms")
    print(f"  speedup            : {(t_load + t_scan) / max(t_cache, 0.01):6.1f}x")
    return 0


def cmd_install_rules(args) -> int:
    """Deploy contrib/rules/ -> ~/.claude/ (authorization doctrine + settings + hook).

    With --no-hook (or when binary patch 05 auto-allow-hook is applied), the
    PreToolUse shell hook is skipped — the binary-level allow decision already
    fires per tool call with zero subprocess spawn overhead.
    """
    src_dir = ROOT / "contrib" / "rules"
    if not src_dir.exists():
        print(f"{R}contrib/rules/ missing in package{X}")
        return 2

    claude_dir = Path.home() / ".claude"
    hooks_dir  = claude_dir / "hooks"
    claude_dir.mkdir(parents=True, exist_ok=True)
    hooks_dir.mkdir(parents=True, exist_ok=True)

    skip_hook = bool(getattr(args, "no_hook", False))
    # Auto-detect: if binary patch 05 is applied, the .sh hook is redundant.
    if not skip_hook:
        try:
            target, kind = find_target()
            if target:
                _text = _scanner.load_cached_rows(target, PATCH_DIR)
                if _text is None:
                    _t = _scanner.load_text_from_target(target, kind)
                    for s in (json.loads((PATCH_DIR / "05-auto-allow-hook.json").read_text(encoding="utf-8")).get("patches") or [{}]):
                        m = s.get("applied_marker")
                        if m and m in _t:
                            skip_hook = True
                            print(f"  {G}{ARROW}{X} binary patch 05 auto-allow detected → skipping .sh hook (faster)")
                            break
        except Exception:
            pass

    auth_src = (src_dir / "AUTHORIZATION.md").read_text(encoding="utf-8")
    block = f"{_VPCC_CLAUDE_MD_START}\n{auth_src.rstrip()}\n{_VPCC_CLAUDE_MD_END}\n"
    import re as _re
    for dst_name in ("CLAUDE.md", "AGENTS.md"):
        dst = claude_dir / dst_name
        if dst.exists():
            existing = dst.read_text(encoding="utf-8")
            if _VPCC_CLAUDE_MD_START in existing:
                existing = _re.sub(
                    f"{_re.escape(_VPCC_CLAUDE_MD_START)}.*?{_re.escape(_VPCC_CLAUDE_MD_END)}\n?",
                    "", existing, flags=_re.DOTALL,
                )
            merged_md = block + "\n---\n\n" + existing.lstrip()
        else:
            merged_md = block
        dst.write_text(merged_md, encoding="utf-8")
        print(f"  {G}{CHECK}{X} {dst_name} {ARROW} {dst}")

    settings_path = claude_dir / "settings.json"
    rules = json.loads((src_dir / "settings-rules.json").read_text(encoding="utf-8"))
    if skip_hook:
        # Drop the auto-allow PreToolUse hook entry from rules before merging
        try:
            pre = rules.get("hooks", {}).get("PreToolUse", [])
            pre = [h for h in pre if "vpcc-auto-allow" not in json.dumps(h)]
            if pre:
                rules["hooks"]["PreToolUse"] = pre
            else:
                rules.get("hooks", {}).pop("PreToolUse", None)
                if not rules.get("hooks"):
                    rules.pop("hooks", None)
        except Exception:
            pass
    current = {}
    if settings_path.exists():
        try:
            current = json.loads(settings_path.read_text(encoding="utf-8"))
        except json.JSONDecodeError:
            bak = settings_path.with_suffix(".json.corrupt.bak")
            settings_path.rename(bak)
            print(f"  {Y}{WARN_ICON} existing settings.json corrupt -> backed up to {bak.name}{X}")
    merged = _deep_merge(current, rules)
    settings_path.write_text(json.dumps(merged, indent=2) + "\n", encoding="utf-8")
    print(f"  {G}{CHECK}{X} settings.json {ARROW} {settings_path} (merged {len(rules)} keys)")

    if skip_hook:
        # Strip any pre-existing auto-allow hook entry from the merged settings
        try:
            pre = merged.get("hooks", {}).get("PreToolUse", [])
            pre = [h for h in pre if "vpcc-auto-allow" not in json.dumps(h)]
            if pre:
                merged["hooks"]["PreToolUse"] = pre
            else:
                merged.get("hooks", {}).pop("PreToolUse", None)
                if not merged.get("hooks"):
                    merged.pop("hooks", None)
            settings_path.write_text(json.dumps(merged, indent=2) + "\n", encoding="utf-8")
        except Exception:
            pass
        # Delete leftover hook file if present
        hook_dst = hooks_dir / "vpcc-auto-allow.sh"
        if hook_dst.exists():
            hook_dst.unlink()
            print(f"  {G}{CHECK}{X} removed redundant .sh hook (binary patch handles allow)")
        else:
            print(f"  {G}{CHECK}{X} no .sh hook installed (zero subprocess overhead per tool call)")
    else:
        hook_src = src_dir / "hooks" / "vpcc-auto-allow.sh"
        hook_dst = hooks_dir / "vpcc-auto-allow.sh"
        hook_dst.write_bytes(hook_src.read_bytes())
        try:
            hook_dst.chmod(0o755)
        except OSError:
            pass  # Windows: chmod not fully supported
        print(f"  {G}{CHECK}{X} hook {ARROW} {hook_dst}")

    print(f"\n{G}{CHECK} authorization rules installed{X}")
    print(f"  revert: vpcc uninstall-rules")
    return 0


def cmd_uninstall_rules(args) -> int:
    """Remove vpcc-authored rules. Operator content preserved."""
    claude_dir = Path.home() / ".claude"
    removed = 0
    import re as _re

    for name in ("CLAUDE.md", "AGENTS.md"):
        p = claude_dir / name
        if not p.exists():
            continue
        s = p.read_text(encoding="utf-8")
        if _VPCC_CLAUDE_MD_START not in s:
            continue
        s2 = _re.sub(
            f"{_re.escape(_VPCC_CLAUDE_MD_START)}.*?{_re.escape(_VPCC_CLAUDE_MD_END)}\n?",
            "", s, flags=_re.DOTALL,
        )
        s2 = _re.sub(r"\A\s*---\s*\n", "", s2)
        p.write_text(s2.lstrip(), encoding="utf-8")
        print(f"  {G}{CHECK}{X} stripped vpcc block from {name}")
        removed += 1

    hook = claude_dir / "hooks" / "vpcc-auto-allow.sh"
    if hook.exists():
        hook.unlink()
        print(f"  {G}{CHECK}{X} removed {hook}")
        removed += 1

    settings_path = claude_dir / "settings.json"
    if settings_path.exists():
        try:
            cur = json.loads(settings_path.read_text(encoding="utf-8"))
        except json.JSONDecodeError:
            cur = {}
        rules_src = ROOT / "contrib" / "rules" / "settings-rules.json"
        if rules_src.exists():
            rules = json.loads(rules_src.read_text(encoding="utf-8"))
            for k in ("_vpcc_header", "skipDangerousModePermissionPrompt",
                      "dangerouslyDisableSandbox", "isBypassPermissionsModeAvailable",
                      "bypassPermissionsWorkspaceTrustCheck", "autoUpdaterStatus"):
                cur.pop(k, None)
            if isinstance(cur.get("env"), dict) and isinstance(rules.get("env"), dict):
                for k in rules["env"]:
                    cur["env"].pop(k, None)
                if not cur["env"]:
                    cur.pop("env")
            try:
                pre = cur.get("hooks", {}).get("PreToolUse", [])
                pre = [h for h in pre if "vpcc-auto-allow" not in json.dumps(h)]
                if pre:
                    cur["hooks"]["PreToolUse"] = pre
                else:
                    cur.get("hooks", {}).pop("PreToolUse", None)
                    if not cur.get("hooks"):
                        cur.pop("hooks", None)
            except Exception:
                pass
            settings_path.write_text(json.dumps(cur, indent=2) + "\n", encoding="utf-8")
            print(f"  {G}{CHECK}{X} pruned vpcc keys from settings.json")
            removed += 1

    if removed == 0:
        print(f"{Y}nothing to remove{X}")
    else:
        print(f"\n{G}{CHECK} authorization rules removed ({removed} item(s)){X}")
    return 0


def cmd_install_preload(args) -> int:
    """Copy contrib/preload/claude-preload.js into wrapper's expected path."""
    src = ROOT / "contrib" / "preload" / "claude-preload.js"
    if not src.exists():
        print(f"{R}source missing: {src}{X}")
        return 2
    if os.name == 'nt':
        dst_dir = Path(os.environ.get("LOCALAPPDATA", str(Path.home() / "AppData" / "Local"))) / "void-patcher"
    else:
        dst_dir = Path.home() / ".local" / "share" / "void-patcher"
    dst_dir.mkdir(parents=True, exist_ok=True)
    dst = dst_dir / "claude-preload.js"
    dst.write_bytes(src.read_bytes())
    try:
        dst.chmod(0o644)
    except OSError:
        pass  # Windows: chmod not fully supported
    print(f"{G}{CHECK} installed preload{X}  {src.name} {ARROW} {dst}")
    print(f"  wrapper will auto-load via BUN_OPTIONS=--preload on next run")
    return 0


def cmd_uninstall_preload(args) -> int:
    if os.name == 'nt':
        dst = Path(os.environ.get("LOCALAPPDATA", str(Path.home() / "AppData" / "Local"))) / "void-patcher" / "claude-preload.js"
    else:
        dst = Path.home() / ".local" / "share" / "void-patcher" / "claude-preload.js"
    if dst.exists():
        dst.unlink()
        print(f"{G}{CHECK} removed{X} {dst}")
    else:
        print(f"{Y}not installed{X}")
    return 0


def cmd_check_updates(args) -> int:
    """Show local vs remote patches commit, no changes."""
    info = _updater.upstream_status(PATCH_DIR)
    print(f"{B}vpcc check-updates{X}")
    print(f"  local commit  : {info['local_commit'] or '(unknown)'}")
    print(f"  remote commit : {info['remote_commit'] or '(unreachable)'}")
    print(f"  local files   : {info['local_files']}")
    if info["drift"]:
        print(f"{Y}{WARN_ICON} update available — run 'vpcc self-update'{X}")
        return 1
    if not info["local_commit"] and info["remote_commit"]:
        print(f"{Y}{WARN_ICON} no sync state — run 'vpcc self-update' to pin current{X}")
        return 1
    if info["remote_commit"]:
        print(f"{G}{CHECK} up to date{X}")
    return 0


# ── entry ─────────────────────────────────────────────────────────────────────

def main() -> int:
    ap = argparse.ArgumentParser(
        prog="vpcc",
        description="Void Patcher for Claude Code — regex-signature patches, cli.js + Bun SEA",
    )
    sub = ap.add_subparsers(dest="cmd", metavar="command")
    p_patch = sub.add_parser("patch", help="Apply all patches")
    p_patch.add_argument("--dry-run", "-n", action="store_true")
    sub.add_parser("verify",   help="Check patches are applied")
    sub.add_parser("rollback", help="Restore from most recent backup")
    sub.add_parser("status",   help="Show install state")
    sub.add_parser("list",     help="List patches in catalog")

    p_su = sub.add_parser("self-update",
        help="Pull latest patches/*.json from GitHub and re-apply")
    p_su.add_argument("--dry-run", "-n", action="store_true")
    p_su.add_argument("--force", "-f", action="store_true",
        help="Sync even if commit hashes match")
    p_su.add_argument("--no-reapply", action="store_true",
        help="Skip re-applying after sync")

    p_ah = sub.add_parser("autoheal",
        help="Detect Claude Code drift; self-update + re-patch if broken")
    p_ah.add_argument("--force", "-f", action="store_true",
        help="Run checks even if CC sha unchanged")
    p_ah.add_argument("--quiet", "-q", action="store_true")

    sub.add_parser("check-updates",
        help="Show if remote patches differ from local")

    sub.add_parser("install-preload",
        help="Deploy runtime monkey-patch preload hook (100%% survival layer)")
    sub.add_parser("uninstall-preload",
        help="Remove runtime preload hook")

    p_ir = sub.add_parser("install-rules",
        help="Deploy operator-authorization bundle (CLAUDE.md + settings + hook)")
    p_ir.add_argument("--no-hook", action="store_true",
        help="Skip PreToolUse shell hook; rely on binary patch 05 (zero subprocess overhead)")
    sub.add_parser("uninstall-rules",
        help="Remove vpcc authorization rules, preserve operator content")

    p_sc = sub.add_parser("scan",
        help="Signature-based offset discovery (survives regex drift)")
    p_sc.add_argument("--verbose", "-v", action="store_true")
    p_sc.add_argument("--export-patch", metavar="ID",
        help="Regenerate regex for patch ID from anchor strings")
    p_sc.add_argument("--auto-heal", action="store_true",
        help="Rewrite drifted regexes in patches/*.json from anchor windows")

    sub.add_parser("doctor",
        help="Full health report: target, patches, sig drift, upstream")

    p_w = sub.add_parser("watch",
        help="Daemon: poll target, autoheal on change")
    p_w.add_argument("--interval", "-i", type=int, default=10,
        help="Poll interval seconds (default 10)")

    sub.add_parser("upgrade",
        help="All-in-one: self-update + autoheal + verify + warm cache")
    sub.add_parser("bench",
        help="Microbenchmark: sha256, text load, scan cold/cached")
    sub.add_parser("guard", help="Fast SHA guard: auto-patch if CC binary changed (<100ms when current)")
    sub.add_parser("install-guard", help="Install platform-specific auto-patch scheduler (Task Scheduler / launchd / systemd)")
    sub.add_parser("uninstall-guard", help="Remove auto-patch scheduler")
    sub.add_parser("autopilot",
        help="Full pipeline: scan → auto-heal → patch → git commit/push → GitHub issue")
    p_dash = sub.add_parser("dashboard",
        help="Real-time status display (version, SHA, patches, guard, drift)")
    p_dash.add_argument("--interval", "-i", type=int, default=5,
        help="Refresh interval seconds (default 5)")
    sub.add_parser("tui",
        help="Interactive terminal UI — full vpcc control panel (curses)")
    sub.add_parser("update",
        help="Full self-update: upgrade vpcc + sync patches + re-patch binary (like omp update)")

    args = ap.parse_args()
    if args.cmd is None:
        ap.print_help()
        return 0
    if args.cmd == "tui":
        from .tui import run_curses_tui
        return run_curses_tui()
    return {"patch": cmd_patch, "verify": cmd_verify,
            "rollback": cmd_rollback, "status": cmd_status,
            "list": cmd_list,
            "self-update": cmd_self_update,
            "autoheal": cmd_autoheal,
            "check-updates": cmd_check_updates,
            "scan": cmd_scan,
            "doctor": cmd_doctor,
            "watch": cmd_watch,
            "install-preload": cmd_install_preload,
            "uninstall-preload": cmd_uninstall_preload,
            "install-rules": cmd_install_rules,
            "uninstall-rules": cmd_uninstall_rules,
            "upgrade": cmd_upgrade,
            "guard": cmd_guard,
            "install-guard": cmd_install_guard,
            "uninstall-guard": cmd_uninstall_guard,
            "autopilot": cmd_autopilot,
            "dashboard": cmd_dashboard,
            "update": cmd_update,
            "bench": cmd_bench}[args.cmd](args)


if __name__ == "__main__":
    sys.exit(main())
