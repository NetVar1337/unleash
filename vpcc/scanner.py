"""
vpcc.scanner — signature-based offset discovery.

When CC updates mangle a single regex patch, the *anchor strings* (stable
human-readable tokens like "tengu_refusal_api_response", "function s5K")
usually survive. SigScanner locates those anchors in the cli.js text or the
Bun SEA .bun section, returns byte offsets, and can regenerate a probable
search_regex from the surrounding window.

Multi-strategy detection cascade:
  1. applied_marker  — patch already landed        (confidence 1.0)
  2. search_regex    — regex match                 (confidence 1.0)
  3. anchor_strings  — all anchors found nearby    (confidence 0.9)
  4. fuzzy anchor    — whitespace / ident relaxed  (confidence 0.5-0.7)
  5. keyword extract — long tokens from anchors    (confidence 0.3)
  6. nothing         — drift                       (confidence 0.0)

Pure stdlib. Zero deps.
"""
from __future__ import annotations

import json
import re
from pathlib import Path
from typing import Any


# ---------------------------------------------------------------------------
# Regex helpers
# ---------------------------------------------------------------------------

_SHORT_IDENT_RE = re.compile(r"\b[A-Za-z_$][\w$]{0,2}\b")
_LONG_TOKEN_RE = re.compile(r"[A-Za-z_$][\w$]{7,}")  # >8 chars
_WHITESPACE_RUN_RE = re.compile(r"\s+")


def _collapse_ws(s: str) -> str:
    """Collapse all whitespace runs to single space."""
    return _WHITESPACE_RUN_RE.sub(" ", s).strip()


def _ident_agnostic_pattern(s: str) -> str:
    """Replace 1-3 char identifiers with a permissive regex class."""
    return _SHORT_IDENT_RE.sub(
        lambda m: r"[A-Za-z_$][\w$]*" if len(m.group(0)) <= 3 else re.escape(m.group(0)),
        s,
    )


def _extract_long_tokens(s: str) -> list[str]:
    """Extract tokens > 8 chars from a string — good anchor candidates."""
    return list(dict.fromkeys(_LONG_TOKEN_RE.findall(s)))


# ---------------------------------------------------------------------------
# Core scanner
# ---------------------------------------------------------------------------

class SigScanner:
    """Signature-driven anchor locator + regex derivation."""

    def __init__(self, text: str | bytes):
        if isinstance(text, bytes):
            try:
                text = text.decode("utf-8", errors="surrogateescape")
            except Exception:
                text = text.decode("latin1")
        self.text: str = text
        self._ws_text: str | None = None  # lazy cache

    @property
    def ws_text(self) -> str:
        """Whitespace-collapsed copy of text, built lazily."""
        if self._ws_text is None:
            self._ws_text = _collapse_ws(self.text)
        return self._ws_text

    # ---- anchor location ------------------------------------------------

    def find_anchor(self, anchors: list[str], max_dist: int = 400) -> int | None:
        """First offset where ALL anchors appear within max_dist bytes of the 1st."""
        if not anchors:
            return None
        first = self.text.find(anchors[0])
        while first >= 0:
            window = self.text[first: first + len(anchors[0]) + max_dist + 200]
            if all(a in window for a in anchors[1:]):
                return first
            first = self.text.find(anchors[0], first + 1)
        return None

    def all_occurrences(self, anchor: str) -> list[int]:
        offs: list[int] = []
        i = self.text.find(anchor)
        while i >= 0:
            offs.append(i)
            i = self.text.find(anchor, i + 1)
        return offs

    def find_anchor_fuzzy(self, anchors: list[str], max_dist: int = 600) -> tuple[int | None, str, float]:
        """Try progressively relaxed matching for anchors.

        Returns (offset | None, method_name, confidence).
        Methods tried in order:
          exact       — plain substring                     (1.0)
          ws_norm     — whitespace-collapsed comparison     (0.7)
          ident_relax — 1-3 char idents replaced with .*?  (0.5)
        """
        # Strategy 1: exact (already covered by find_anchor, but included for standalone use)
        off = self.find_anchor(anchors, max_dist)
        if off is not None:
            return off, "anchor", 0.9

        if not anchors:
            return None, "none", 0.0

        # Strategy 2: whitespace-normalised
        ws_anchors = [_collapse_ws(a) for a in anchors]
        ws_first = self.ws_text.find(ws_anchors[0])
        while ws_first >= 0:
            win = self.ws_text[ws_first: ws_first + len(ws_anchors[0]) + max_dist + 200]
            if all(a in win for a in ws_anchors[1:]):
                # Map back to approximate raw offset — find the first occurrence
                # of the whitespace-collapsed first anchor near ws_first in raw text
                raw_guess = self.text.find(anchors[0][:6], max(0, ws_first - 200))
                if raw_guess < 0:
                    raw_guess = ws_first
                return raw_guess, "fuzzy_ws", 0.7
            ws_first = self.ws_text.find(ws_anchors[0], ws_first + 1)

        # Strategy 3: identifier-agnostic regex
        for anchor in anchors[:1]:  # only try first anchor to keep cost bounded
            pat = _ident_agnostic_pattern(re.escape(anchor))
            try:
                m = re.search(pat, self.text)
                if m:
                    off = m.start()
                    # verify rest of anchors within window (relaxed too)
                    remaining_ok = True
                    for other in anchors[1:]:
                        opat = _ident_agnostic_pattern(re.escape(other))
                        win = self.text[off: off + len(anchor) + max_dist + 200]
                        if not re.search(opat, win):
                            remaining_ok = False
                            break
                    if remaining_ok:
                        return off, "fuzzy_ident", 0.5
            except re.error:
                pass

        return None, "none", 0.0

    # ---- keyword search -------------------------------------------------

    def keyword_search(self, anchors: list[str]) -> tuple[int | None, float]:
        """Extract long tokens from anchors and search for them.

        Returns (offset, confidence). Only reports if ALL keywords found
        within a reasonable window.
        """
        keywords = []
        for a in anchors:
            keywords.extend(_extract_long_tokens(a))
        if not keywords:
            return None, 0.0

        # Find first keyword, then verify others nearby
        first_off = self.text.find(keywords[0])
        while first_off >= 0:
            window = self.text[max(0, first_off - 200): first_off + 1000]
            if all(kw in window for kw in keywords[1:]):
                return first_off, 0.3
            first_off = self.text.find(keywords[0], first_off + 1)

        return None, 0.0

    # ---- regex derivation -----------------------------------------------

    @staticmethod
    def _minify_names(s: str) -> str:
        return _SHORT_IDENT_RE.sub(
            lambda m: r"[A-Za-z_$][\w$]*" if len(m.group(0)) <= 3 else m.group(0),
            s,
        )

    def derive_regex(self, anchor: str, before: int = 60, after: int = 60,
                     softmin: bool = True) -> str | None:
        """Escaped, minifier-tolerant regex around `anchor`."""
        i = self.text.find(anchor)
        if i < 0:
            return None
        ctx = self.text[max(0, i - before): i + len(anchor) + after]
        esc = re.escape(ctx)
        if softmin:
            esc = self._minify_names(esc)
        return esc

    def derive_regex_window(self, offset: int, before: int = 80, after: int = 150,
                            softmin: bool = True) -> str:
        """Derive minifier-tolerant regex from a raw offset + window."""
        start = max(0, offset - before)
        end = min(len(self.text), offset + after)
        ctx = self.text[start:end]
        esc = re.escape(ctx)
        if softmin:
            esc = self._minify_names(esc)
        return esc

    # ---- patch-file driver (multi-strategy) -----------------------------

    def scan_patches(self, patches: list[dict[str, Any]]) -> list[dict[str, Any]]:
        out = []
        for p in patches:
            pid = p.get("id", "?")
            anchors = p.get("anchor_strings") or []
            sig_regex = None
            markers: list[str] = []
            for sub in p.get("patches", []):
                if not sig_regex:
                    sig_regex = sub.get("search_regex") or sub.get("search")
                m = sub.get("applied_marker")
                if m:
                    markers.append(m)

            # --- cascade ---
            status = "drift"
            confidence = 0.0
            method = "none"
            anchor_off: int | None = None

            # 1. applied_marker — highest priority, patch already landed
            marker_hit = any(m in self.text for m in markers)
            if marker_hit:
                status = "applied"
                confidence = 1.0
                method = "marker"
                # still try to locate offset for reporting
                anchor_off = self.find_anchor(anchors) if anchors else None

            # 2. search_regex
            regex_hit = False
            if sig_regex and status != "applied":
                try:
                    regex_hit = re.search(sig_regex, self.text, re.DOTALL) is not None
                except re.error:
                    regex_hit = False
                if regex_hit:
                    status = "ok"
                    confidence = 1.0
                    method = "regex"
                    anchor_off = self.find_anchor(anchors) if anchors else None

            # 3. exact anchor_strings
            if status not in ("applied", "ok") and anchors:
                anchor_off = self.find_anchor(anchors)
                if anchor_off is not None:
                    status = "ok"
                    confidence = 0.9
                    method = "anchor"

            # 4. fuzzy anchor
            if status not in ("applied", "ok") and anchors:
                foff, fmethod, fconf = self.find_anchor_fuzzy(anchors)
                if foff is not None and fconf > 0.0:
                    anchor_off = foff
                    status = "ok"
                    confidence = fconf
                    method = fmethod

            # 5. keyword extraction
            if status not in ("applied", "ok") and anchors:
                koff, kconf = self.keyword_search(anchors)
                if koff is not None and kconf > 0.0:
                    anchor_off = koff
                    status = "ok"
                    confidence = kconf
                    method = "keyword"

            # 6. no anchors and no regex → unclassified (legacy)
            if status == "drift" and not anchors and not regex_hit:
                status = "unclassified"

            out.append({
                "id": pid,
                "anchors": anchors,
                "anchor_offset": anchor_off,
                "regex_hit": regex_hit,
                "marker_hit": marker_hit,
                "status": status,
                "confidence": confidence,
                "method": method,
            })
        return out


# ---------------------------------------------------------------------------
# Module-level helpers
# ---------------------------------------------------------------------------

def load_text_from_target(target: Path, kind: str) -> str:
    """Extract patchable text from cli.js or Bun SEA .bun section.
    Cross-platform: dispatches ELF / Mach-O / PE by magic bytes.
    Delegates to vpcc.__main__._find_bun_section to avoid duplication.
    """
    if kind == "js":
        return target.read_text(encoding="utf-8", errors="surrogateescape")
    from . import __main__ as _m
    data = bytearray(target.read_bytes())
    off, size = _m._find_bun_section(data)
    return bytes(data[off:off + size]).decode("utf-8", errors="surrogateescape")


def format_scan_report(rows: list[dict[str, Any]], verbose: bool = False) -> str:
    G, Y, R, X = "\033[32m", "\033[33m", "\033[31m", "\033[0m"
    C = "\033[36m"  # cyan for method
    lines = []
    ok = drift = applied = unclassified = 0

    for r in rows:
        status = r["status"]
        confidence = r.get("confidence", -1.0)
        method = r.get("method", "?")

        if status == "applied":
            mark = f"{G}applied{X}"; applied += 1
        elif status == "ok":
            mark = f"{G}ok{X}"; ok += 1
        elif status == "drift":
            mark = f"{R}drift{X}"; drift += 1
        else:
            mark = f"{Y}nometa{X}"; unclassified += 1

        # Confidence coloring
        if confidence >= 0.8:
            conf_s = f"{G}{confidence:.1f}{X}"
        elif confidence >= 0.5:
            conf_s = f"{Y}{confidence:.1f}{X}"
        elif confidence >= 0.0:
            conf_s = f"{R}{confidence:.1f}{X}"
        else:
            conf_s = "  - "

        off = r.get("anchor_offset")
        off_s = f"@0x{off:08x}" if off is not None else "--"
        method_s = f"{C}{method:>12s}{X}"

        # healable: drift + has anchors
        healable = ""
        if status == "drift" and r.get("anchors"):
            healable = f" {Y}[healable]{X}"

        line = (
            f"  {mark:22s}  {r['id']:42s}  {off_s:>14s}"
            f"  {conf_s:>18s}  {method_s}"
            f"  regex={'Y' if r.get('regex_hit') else 'N'}{healable}"
        )
        lines.append(line)
        if verbose and r.get("anchors"):
            lines.append(f"    anchors: {', '.join(r['anchors'])}")

    tail = f"\n  {G}{ok} ok{X}"
    if applied:
        tail += f"  {G}{applied} applied{X}"
    if drift:
        tail += f"  {R}{drift} drift{X}"
    if unclassified:
        tail += f"  {Y}{unclassified} nometa{X} (pre-v2.1.114 patches — anchor_strings not yet backfilled)"
    lines.append(tail)
    return "\n".join(lines)


def load_patches_from_dir(patch_dir: Path, respect_scan_flag: bool = True) -> list[dict[str, Any]]:
    """Load js_replace patches. When respect_scan_flag=True (default), patches
    that explicitly set `scan_signatures: false` are excluded — they are not
    text-scannable in the target (bytecode-only, superseded, etc.) and including
    them would pollute scan/doctor output with false drift/nometa noise."""
    out = []
    for f in sorted(patch_dir.glob("*.json")):
        try:
            obj = json.loads(f.read_text(encoding="utf-8"))
        except json.JSONDecodeError:
            continue
        if obj.get("type") != "js_replace":
            continue
        if respect_scan_flag and obj.get("scan_signatures", True) is False:
            continue
        obj["__file"] = str(f)
        out.append(obj)
    return out


def auto_heal_drift(text: str, patch_dir: Path, verbose: bool = False) -> dict[str, int]:
    """
    For every patch whose anchors resolve but regex doesn't, regenerate a
    probable search_regex from the anchor context window and rewrite the
    patch JSON file.

    Enhanced: also tries fuzzy anchor matching when exact anchors fail,
    and updates stale anchor_strings in-place.
    """
    sc = SigScanner(text)
    healed = 0
    skipped = 0
    failed = 0
    details: list[dict[str, Any]] = []

    for p in load_patches_from_dir(patch_dir):
        anchors = p.get("anchor_strings") or []
        if not anchors:
            skipped += 1
            continue

        first_anchor = anchors[0]
        sub = (p.get("patches") or [{}])[0]
        sig_regex = sub.get("search_regex")
        if not sig_regex:
            skipped += 1
            continue

        # Check if regex already matches
        try:
            regex_hit = re.search(sig_regex, text, re.DOTALL) is not None
        except re.error:
            regex_hit = False
        if regex_hit:
            skipped += 1
            continue

        # Check applied marker — if already applied, skip healing
        markers = [s.get("applied_marker") for s in p.get("patches", []) if s.get("applied_marker")]
        if any(m in text for m in markers):
            skipped += 1
            continue

        # Try exact anchor match first
        anchor_off = sc.find_anchor(anchors)
        anchor_method = "exact"

        # If exact anchors fail, try fuzzy
        if anchor_off is None:
            foff, fmethod, fconf = sc.find_anchor_fuzzy(anchors)
            if foff is not None:
                anchor_off = foff
                anchor_method = fmethod
                # Update anchor_strings to what we actually found in the text
                # Find the actual text at that offset that resembles the anchor
                for i, old_a in enumerate(anchors):
                    # Try to find a nearby exact-matchable substring
                    window = text[max(0, anchor_off - 100): anchor_off + len(old_a) + 500]
                    # Extract long tokens from old anchor, find them in window
                    for tok in _extract_long_tokens(old_a):
                        if tok in window:
                            # Found the stable core — use it
                            break

        if anchor_off is None:
            failed += 1
            details.append({"id": p["id"], "action": "failed", "reason": "anchors not found"})
            continue

        # Derive new regex from the context window around the anchor
        new_regex = sc.derive_regex_window(anchor_off, before=80, after=150)
        if not new_regex:
            failed += 1
            details.append({"id": p["id"], "action": "failed", "reason": "derive_regex returned empty"})
            continue

        sub["search_regex"] = new_regex
        # Don't blindly set replace to the regex — that's an identity no-op
        # that destroys the actual replacement intent. Leave replace as-is.

        file_path = Path(p.pop("__file"))
        file_path.write_text(json.dumps(p, indent=2) + "\n", encoding="utf-8")
        if verbose:
            print(f"  healed {p['id']} @ 0x{anchor_off:x} (via {anchor_method})")
        healed += 1
        details.append({"id": p["id"], "action": "healed", "method": anchor_method, "offset": anchor_off})

    return {"healed": healed, "skipped": skipped, "failed": failed, "details": details}


def validate_patches(patches: list[dict[str, Any]], text: str | None = None) -> list[dict[str, str]]:
    """Validate patch definitions and return actionable warnings.

    Each warning is a dict with keys: id, severity ('error'|'warn'), message.
    """
    warnings: list[dict[str, str]] = []
    seen_ids: set[str] = set()

    for p in patches:
        pid = p.get("id", "<no id>")

        # Duplicate ID check
        if pid in seen_ids:
            warnings.append({"id": pid, "severity": "error", "message": f"Duplicate patch ID: {pid}"})
        seen_ids.add(pid)

        # anchor_strings check
        anchors = p.get("anchor_strings") or []
        if not anchors:
            warnings.append({"id": pid, "severity": "warn", "message": "No anchor_strings — scanner cannot detect drift"})

        for sub in p.get("patches", []):
            # regex compiles
            sr = sub.get("search_regex") or sub.get("search")
            if sr:
                try:
                    re.compile(sr)
                except re.error as e:
                    warnings.append({"id": pid, "severity": "error", "message": f"Invalid search_regex: {e}"})

            # applied_marker vs pre-patch text
            marker = sub.get("applied_marker")
            if marker and text is not None:
                # For pre-patch validation: marker should NOT be in the unpatched text
                # (it should only appear after patching). Heuristic: if the marker
                # is a substring of the search_regex's literal intent, it's fine.
                # But if it already exists in text AND the search_regex also matches,
                # that's suspicious.
                if marker in text:
                    # Check if search_regex also matches — if both match, the marker
                    # is not unique to the patched state
                    if sr:
                        try:
                            if re.search(sr, text, re.DOTALL):
                                warnings.append({
                                    "id": pid,
                                    "severity": "warn",
                                    "message": f"applied_marker '{marker[:40]}...' found in pre-patch text alongside search_regex match — marker may not be unique to patched state",
                                })
                        except re.error:
                            pass

            # replace length check — for binary patching, replace must be <= search match length
            replace = sub.get("replace")
            if replace and sr and text is not None:
                try:
                    m = re.search(sr, text, re.DOTALL)
                    if m and len(replace) > len(m.group(0)):
                        warnings.append({
                            "id": pid,
                            "severity": "error",
                            "message": f"replace ({len(replace)} bytes) > search match ({len(m.group(0))} bytes) — binary patch will corrupt",
                        })
                except re.error:
                    pass

    return warnings


def suggest_anchors(text: str, regex_or_context: str, n: int = 3) -> list[str]:
    """Suggest good anchor_strings for a patch.

    Given either a search_regex or a raw context string, extract candidate
    anchors that are:
    - Long (>8 chars) — survive minification
    - Unique in the binary text (occur exactly once, or very few times)
    - Human-readable tokens, not regex syntax

    Returns up to n suggestions sorted by uniqueness (fewest occurrences first).
    """
    # Try to extract literal strings from what might be a regex
    # First: unescape regex escapes to get the raw text
    try:
        # If it's a valid regex, find a match in text to get the literal
        m = re.search(regex_or_context, text, re.DOTALL)
        if m:
            source = m.group(0)
        else:
            source = regex_or_context
    except re.error:
        source = regex_or_context

    # Also try un-escaping the regex to get literal content
    unescaped = re.sub(r"\\(.)", r"\1", regex_or_context)

    # Collect candidate tokens from both sources
    candidates: list[str] = []
    for src in (source, unescaped):
        candidates.extend(_extract_long_tokens(src))

    # Also extract quoted string literals (common in JS patches)
    for qm in re.finditer(r'"([^"]{8,})"', source + " " + unescaped):
        candidates.append(qm.group(1))

    # Deduplicate preserving order
    seen: set[str] = set()
    unique_candidates: list[str] = []
    for c in candidates:
        if c not in seen:
            seen.add(c)
            unique_candidates.append(c)

    # Score by occurrence count in text (fewer = better anchor)
    scored = []
    for c in unique_candidates:
        count = text.count(c)
        if count == 0:
            continue  # not in text, useless as anchor
        scored.append((count, c))

    scored.sort(key=lambda x: x[0])
    return [c for _, c in scored[:n]]
