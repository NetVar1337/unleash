"""
vpcc interactive TUI — curses-based terminal interface.

Cross-platform: uses curses on macOS/Linux, windows-curses on Windows.
Falls back to ANSI refresh loop if curses unavailable.

Launched via: vpcc tui
"""
from __future__ import annotations

import os
import sys
import time
import subprocess
import threading
from pathlib import Path
from datetime import datetime
from typing import Any

# ── lazy imports from vpcc core (avoid circular at module level) ─────────────
def _core():
    from vpcc import __main__ as m
    return m

def _scanner():
    from vpcc import scanner as s
    return s


# ── data model ───────────────────────────────────────────────────────────────

class State:
    """Mutable shared state refreshed by background thread."""
    __slots__ = (
        "target", "kind", "cc_version", "sha", "size_mb", "format_str",
        "guard_status", "guard_age", "scheduler",
        "n_ok", "n_applied", "n_drift", "n_retired", "drift_ids",
        "n_backups", "upstream",
        "last_refresh", "refreshing", "error",
        "action_log",
    )

    def __init__(self):
        self.target: str = ""
        self.kind: str = ""
        self.cc_version: str = ""
        self.sha: str = ""
        self.size_mb: int = 0
        self.format_str: str = ""
        self.guard_status: str = "unknown"
        self.guard_age: str = ""
        self.scheduler: str = "unknown"
        self.n_ok = self.n_applied = self.n_drift = self.n_retired = 0
        self.drift_ids: list[str] = []
        self.n_backups: int = 0
        self.upstream: str = "unknown"
        self.last_refresh: str = ""
        self.refreshing: bool = False
        self.error: str = ""
        self.action_log: list[str] = []

    def log(self, msg: str):
        ts = datetime.now().strftime("%H:%M:%S")
        self.action_log.append(f"[{ts}] {msg}")
        if len(self.action_log) > 50:
            self.action_log = self.action_log[-50:]


def refresh_state(state: State):
    """Refresh all state fields from live system. Thread-safe on fields."""
    state.refreshing = True
    state.error = ""
    m = _core()
    try:
        target, kind = m.find_target()
        if not target:
            state.target = ""
            state.error = "Claude Code not found"
            state.refreshing = False
            return

        state.target = str(target)
        state.kind = kind
        state.sha = m.sha256_short(target)
        state.size_mb = target.stat().st_size // (1024 * 1024)
        state.format_str = "Bun SEA" if kind == "bun_sea" else "cli.js"
        state.cc_version = m._detect_cc_version(target)

        # Guard stamp
        stamp = Path.home() / ".vpcc" / "last_patched_sha"
        if stamp.is_file():
            stamp_sha = stamp.read_text().strip()
            age = time.time() - stamp.stat().st_mtime
            state.guard_age = m._human_age(age)
            state.guard_status = "patched" if stamp_sha == state.sha else "stale"
        else:
            state.guard_status = "none"
            state.guard_age = ""

        # Scheduler
        state.scheduler = _detect_scheduler_short()

        # Scan cache
        sc = _scanner()
        cached = sc.load_cached_rows(target, m.PATCH_DIR)
        if cached:
            state.n_ok = sum(1 for r in cached if r["status"] == "ok")
            state.n_applied = sum(1 for r in cached if r["status"] == "applied")
            state.n_drift = sum(1 for r in cached if r["status"] == "drift")
            state.n_retired = sum(1 for r in cached if r["status"] == "retired")
            state.drift_ids = [r["id"] for r in cached if r["status"] == "drift"]
        else:
            state.n_ok = state.n_applied = state.n_drift = state.n_retired = -1
            state.drift_ids = []

        # Backups
        state.n_backups = len(list(m.BACKUP_DIR.glob("claude.*.bak")))

        # Upstream
        try:
            from vpcc import updater as _upd
            info = _upd.upstream_status(m.PATCH_DIR)
            if info["drift"]:
                state.upstream = "behind"
            elif info["remote_commit"]:
                state.upstream = "current"
            else:
                state.upstream = "unreachable"
        except Exception:
            state.upstream = "error"

        state.last_refresh = datetime.now().strftime("%H:%M:%S")
    except Exception as e:
        state.error = str(e)[:60]
    finally:
        state.refreshing = False


def _detect_scheduler_short() -> str:
    if sys.platform == "win32":
        try:
            r = subprocess.run(
                ["schtasks", "/Query", "/TN", "vpcc-autoheal", "/FO", "LIST"],
                capture_output=True, text=True, timeout=3)
            return "active" if r.returncode == 0 else "off"
        except Exception:
            return "off"
    elif sys.platform == "darwin":
        p = Path.home() / "Library" / "LaunchAgents" / "cc.voidchecksum.vpcc-guard.plist"
        return "active" if p.exists() else "off"
    else:
        try:
            r = subprocess.run(
                ["systemctl", "--user", "is-active", "vpcc-guard.timer"],
                capture_output=True, text=True, timeout=3)
            return "active" if r.stdout.strip() == "active" else "off"
        except Exception:
            return "off"


# ── menu actions ─────────────────────────────────────────────────────────────

MENU_ITEMS = [
    ("Update vpcc",         "update",         "Full self-update: upgrade vpcc + patches + re-patch (like omp update)"),
    ("Patch binary",        "patch",          "Apply all patches to Claude Code binary"),
    ("Verify patches",      "verify",         "Check all applied markers are present"),
    ("Full autopilot",      "autopilot",      "Scan → heal → patch → commit → push"),
    ("Scan signatures",     "scan",           "Signature scan with drift detection"),
    ("Doctor health check", "doctor",         "Full health report"),
    ("Install rules",       "install-rules",  "Deploy authorization bundle (--no-hook)"),
    ("Install guard",       "install-guard",  "Set up auto-patch scheduler"),
    ("Upgrade (all-in-one)","upgrade",        "Self-update patches + autoheal + verify + cache"),
    ("Rollback",            "rollback",       "Restore from most recent backup"),
    ("Scan auto-heal",      "scan-heal",      "Scan + regenerate drifted regexes"),
]


def run_action(cmd: str, state: State) -> int:
    """Execute a vpcc action, capturing output into state.action_log."""
    m = _core()
    state.log(f">>> vpcc {cmd}")

    class Args:
        dry_run = False
        verbose = False
        force = False
        quiet = False
        no_hook = True
        export_patch = None
        auto_heal = False
        interval = 10
        no_reapply = False

    try:
        if cmd == "update":
            rc = m.cmd_update(Args())
        elif cmd == "patch":
            rc = m.cmd_patch(Args())
        elif cmd == "verify":
            rc = m.cmd_verify(Args())
        elif cmd == "autopilot":
            rc = m.cmd_autopilot(Args())
        elif cmd == "scan":
            rc = m.cmd_scan(Args())
        elif cmd == "doctor":
            rc = m.cmd_doctor(Args())
        elif cmd == "install-rules":
            rc = m.cmd_install_rules(Args())
        elif cmd == "install-guard":
            rc = m.cmd_install_guard(Args())
        elif cmd == "upgrade":
            rc = m.cmd_upgrade(Args())
        elif cmd == "rollback":
            rc = m.cmd_rollback(Args())
        elif cmd == "scan-heal":
            Args.auto_heal = True
            Args.verbose = True
            rc = m.cmd_scan(Args())
        else:
            state.log(f"unknown command: {cmd}")
            return 1
        state.log(f"<<< exit {rc}")
        return rc
    except Exception as e:
        state.log(f"ERROR: {e}")
        return 2


# ── curses TUI ───────────────────────────────────────────────────────────────

def _try_curses():
    """Try to import curses. Returns module or None."""
    try:
        import curses
        return curses
    except ImportError:
        pass
    # Windows: try windows-curses
    try:
        import curses
        return curses
    except ImportError:
        return None


def run_curses_tui():
    """Main curses TUI entry point."""
    curses = _try_curses()
    if curses is None:
        print("curses not available — install 'windows-curses' on Windows")
        print("  pip install windows-curses")
        print("\nFalling back to simple ANSI mode...")
        return run_ansi_tui()

    return curses.wrapper(_curses_main)


def _curses_main(stdscr):
    curses = _try_curses()
    curses.curs_set(0)
    stdscr.nodelay(False)
    stdscr.timeout(1000)  # refresh every 1s

    # Colors
    curses.start_color()
    curses.use_default_colors()
    curses.init_pair(1, curses.COLOR_GREEN, -1)
    curses.init_pair(2, curses.COLOR_YELLOW, -1)
    curses.init_pair(3, curses.COLOR_RED, -1)
    curses.init_pair(4, curses.COLOR_CYAN, -1)
    curses.init_pair(5, curses.COLOR_WHITE, curses.COLOR_BLUE)  # selected
    curses.init_pair(6, curses.COLOR_WHITE, -1)  # dim

    CP_GREEN = curses.color_pair(1)
    CP_YELLOW = curses.color_pair(2)
    CP_RED = curses.color_pair(3)
    CP_CYAN = curses.color_pair(4)
    CP_SEL = curses.color_pair(5) | curses.A_BOLD
    CP_DIM = curses.color_pair(6)
    CP_BOLD = curses.A_BOLD

    state = State()
    selected = 0
    mode = "menu"  # "menu" or "action"
    action_scroll = 0
    last_bg_refresh = 0.0

    # Initial refresh
    refresh_state(state)

    while True:
        h, w = stdscr.getmaxyx()
        stdscr.erase()

        # ── Header ────────────────────────────────────────────────────────
        title = " VPCC — Void Patcher for Claude Code "
        stdscr.addstr(0, 0, " " * w, CP_SEL)
        stdscr.addstr(0, max(0, (w - len(title)) // 2), title, CP_SEL)

        ts = state.last_refresh or "..."
        r_label = f" {ts} "
        if w > len(r_label) + 2:
            stdscr.addstr(0, w - len(r_label) - 1, r_label, CP_SEL)

        # ── Status panel (left) ───────────────────────────────────────────
        y = 2
        panel_w = min(42, w // 2 - 1)

        def sline(row, label, value, color=0):
            if row >= h - 1:
                return
            lbl = f"  {label:10s}: "
            stdscr.addstr(row, 0, lbl, CP_BOLD)
            stdscr.addnstr(row, len(lbl), str(value), panel_w - len(lbl), color)

        if state.error:
            stdscr.addstr(y, 2, state.error, CP_RED)
            y += 1
        else:
            sline(y, "target", Path(state.target).name if state.target else "not found",
                  CP_GREEN if state.target else CP_RED); y += 1
            sline(y, "version", state.cc_version or "?", CP_CYAN); y += 1
            sline(y, "sha256", state.sha or "?"); y += 1
            sline(y, "size", f"{state.size_mb} MB" if state.size_mb else "?"); y += 1
            sline(y, "format", state.format_str or "?"); y += 1

            y += 1
            if state.guard_status == "patched":
                sline(y, "guard", f"patched ({state.guard_age} ago)", CP_GREEN)
            elif state.guard_status == "stale":
                sline(y, "guard", "STALE — binary changed", CP_RED)
            else:
                sline(y, "guard", "no stamp", CP_YELLOW)
            y += 1

            sline(y, "scheduler", state.scheduler,
                  CP_GREEN if state.scheduler == "active" else CP_YELLOW); y += 1

            y += 1
            if state.n_ok >= 0:
                sline(y, "patches", f"{state.n_ok + state.n_applied + state.n_drift} total"); y += 1
                status_parts = []
                stdscr.addstr(y, 4, f"{state.n_ok} ok", CP_GREEN)
                stdscr.addstr(y, 4 + len(f"{state.n_ok} ok") + 2,
                              f"{state.n_applied} applied", CP_GREEN)
                col = 4 + len(f"{state.n_ok} ok") + 2 + len(f"{state.n_applied} applied") + 2
                if state.n_drift:
                    stdscr.addstr(y, col, f"{state.n_drift} DRIFT", CP_RED | curses.A_BOLD)
                else:
                    stdscr.addstr(y, col, "0 drift", CP_DIM)
                y += 1

                for did in state.drift_ids[:3]:
                    if y < h - 1:
                        stdscr.addnstr(y, 6, f"! {did}", panel_w - 6, CP_RED)
                        y += 1
            else:
                sline(y, "patches", "(run scan to populate)", CP_DIM); y += 1

            y += 1
            sline(y, "backups", str(state.n_backups)); y += 1
            sline(y, "upstream", state.upstream,
                  CP_GREEN if state.upstream == "current" else
                  CP_YELLOW if state.upstream == "behind" else CP_DIM); y += 1

        # ── Menu panel (right) ────────────────────────────────────────────
        menu_x = max(panel_w + 2, w // 2)
        menu_w = w - menu_x - 1
        if menu_w < 20:
            menu_x = 2
            menu_w = w - 4

        stdscr.addstr(2, menu_x, " Actions ", CP_BOLD)
        stdscr.addstr(2, menu_x + 10, "(j/k or arrows, Enter to run, q to quit)", CP_DIM)

        if mode == "menu":
            for i, (label, cmd, desc) in enumerate(MENU_ITEMS):
                row = 4 + i
                if row >= h - 2:
                    break
                if i == selected:
                    stdscr.addstr(row, menu_x, f" > {label:<28s}", CP_SEL)
                else:
                    stdscr.addstr(row, menu_x, f"   {label:<28s}", CP_CYAN)

            # Description of selected item
            if selected < len(MENU_ITEMS):
                desc_row = 4 + min(len(MENU_ITEMS), h - 6) + 1
                if desc_row < h - 1:
                    desc = MENU_ITEMS[selected][2]
                    stdscr.addnstr(desc_row, menu_x, desc, menu_w, CP_DIM)

        elif mode == "action":
            # Show action log
            stdscr.addstr(3, menu_x, " Output (press q/Esc to return) ", CP_BOLD)
            log_lines = state.action_log
            visible = h - 6
            start = max(0, len(log_lines) - visible - action_scroll)
            end = start + visible
            for i, line in enumerate(log_lines[start:end]):
                row = 5 + i
                if row >= h - 1:
                    break
                # Strip ANSI codes for curses display
                clean = _strip_ansi(line)
                color = CP_RED if "ERROR" in clean or "fail" in clean.lower() else \
                        CP_GREEN if "ok" in clean.lower() or "applied" in clean.lower() else 0
                stdscr.addnstr(row, menu_x, clean, menu_w, color)

        # ── Footer ────────────────────────────────────────────────────────
        footer = " q:quit  r:refresh  u:update  p:patch  a:autopilot  d:doctor "
        if h > 1:
            stdscr.addstr(h - 1, 0, " " * w, CP_SEL)
            stdscr.addnstr(h - 1, 0, footer, w, CP_SEL)

        if state.refreshing:
            stdscr.addstr(h - 1, w - 14, " refreshing ", CP_YELLOW)

        stdscr.refresh()

        # ── Background refresh every 10s ──────────────────────────────────
        now = time.time()
        if now - last_bg_refresh > 10 and not state.refreshing:
            last_bg_refresh = now
            t = threading.Thread(target=refresh_state, args=(state,), daemon=True)
            t.start()

        # ── Input ─────────────────────────────────────────────────────────
        try:
            ch = stdscr.getch()
        except Exception:
            ch = -1

        if ch == -1:
            continue

        if ch == ord("q") or ch == 27:  # q or Esc
            if mode == "action":
                mode = "menu"
                action_scroll = 0
            else:
                return 0

        elif mode == "menu":
            if ch == curses.KEY_UP or ch == ord("k"):
                selected = (selected - 1) % len(MENU_ITEMS)
            elif ch == curses.KEY_DOWN or ch == ord("j"):
                selected = (selected + 1) % len(MENU_ITEMS)
            elif ch in (curses.KEY_ENTER, 10, 13):
                # Run the selected action
                _, cmd, _ = MENU_ITEMS[selected]
                mode = "action"
                action_scroll = 0
                # Capture stdout during action
                state.log(f"--- running: {cmd} ---")
                stdscr.nodelay(True)
                stdscr.timeout(100)
                _run_action_captured(cmd, state)
                # Refresh after action
                refresh_state(state)
                stdscr.nodelay(False)
                stdscr.timeout(1000)

            # Hotkeys
            elif ch == ord("r"):
                state.log("manual refresh")
                refresh_state(state)
            elif ch == ord("p"):
                mode = "action"
                _run_action_captured("patch", state)
                refresh_state(state)
            elif ch == ord("a"):
                mode = "action"
                _run_action_captured("autopilot", state)
                refresh_state(state)
            elif ch == ord("d"):
                mode = "action"
                _run_action_captured("doctor", state)
                refresh_state(state)
            elif ch == ord("u"):
                mode = "action"
                _run_action_captured("update", state)
                refresh_state(state)
        elif mode == "action":
            if ch == curses.KEY_UP or ch == ord("k"):
                action_scroll = min(action_scroll + 1, max(0, len(state.action_log) - 5))
            elif ch == curses.KEY_DOWN or ch == ord("j"):
                action_scroll = max(0, action_scroll - 1)


def _run_action_captured(cmd: str, state: State):
    """Run an action, capturing its stdout/stderr into state.action_log."""
    import io
    old_out, old_err = sys.stdout, sys.stderr
    buf = io.StringIO()
    try:
        sys.stdout = buf
        sys.stderr = buf
        run_action(cmd, state)
    finally:
        sys.stdout = old_out
        sys.stderr = old_err
    for line in buf.getvalue().splitlines():
        clean = _strip_ansi(line)
        if clean.strip():
            state.action_log.append(clean)


def _strip_ansi(s: str) -> str:
    """Remove ANSI escape sequences from a string."""
    import re
    return re.sub(r"\x1b\[[0-9;]*m", "", s)


# ── ANSI fallback TUI ────────────────────────────────────────────────────────

def run_ansi_tui():
    """Simple ANSI-based TUI for terminals without curses support."""
    state = State()
    refresh_state(state)

    CLEAR = "\033[2J\033[H"
    BOLD = "\033[1m"
    GREEN = "\033[32m"
    YELLOW = "\033[33m"
    RED = "\033[31m"
    CYAN = "\033[36m"
    DIM = "\033[90m"
    RESET = "\033[0m"

    print(f"{CLEAR}{BOLD}VPCC — Void Patcher for Claude Code{RESET}")
    print(f"{DIM}{'─' * 50}{RESET}\n")

    if state.error:
        print(f"  {RED}{state.error}{RESET}\n")

    print(f"  {BOLD}target{RESET}   : {state.target or 'not found'}")
    print(f"  {BOLD}version{RESET}  : {state.cc_version}")
    print(f"  {BOLD}sha256{RESET}   : {state.sha}")
    print(f"  {BOLD}guard{RESET}    : {state.guard_status} {state.guard_age}")
    if state.n_ok >= 0:
        print(f"  {BOLD}patches{RESET}  : {GREEN}{state.n_ok} ok{RESET}  "
              f"{GREEN}{state.n_applied} applied{RESET}  "
              f"{RED if state.n_drift else DIM}{state.n_drift} drift{RESET}")
    print()

    menu = [
        ("1", "patch",          "Patch binary"),
        ("2", "verify",         "Verify patches"),
        ("3", "autopilot",      "Full autopilot"),
        ("4", "scan",           "Scan signatures"),
        ("5", "doctor",         "Doctor health check"),
        ("6", "install-rules",  "Install rules (--no-hook)"),
        ("7", "install-guard",  "Install guard scheduler"),
        ("8", "upgrade",        "Full upgrade"),
        ("9", "rollback",       "Rollback to backup"),
        ("0", "scan-heal",      "Scan + auto-heal"),
        ("q", None,             "Quit"),
    ]

    for key, cmd, label in menu:
        print(f"  {CYAN}[{key}]{RESET} {label}")
    print()

    while True:
        try:
            choice = input(f"  {BOLD}>{RESET} ").strip().lower()
        except (EOFError, KeyboardInterrupt):
            print()
            return 0

        if choice == "q":
            return 0

        matched = next((m for m in menu if m[0] == choice and m[1]), None)
        if matched:
            print(f"\n  {BOLD}Running: vpcc {matched[1]}{RESET}\n")
            run_action(matched[1], state)
            print()
            refresh_state(state)
            # Redisplay status
            print(f"\n  {BOLD}guard{RESET}: {state.guard_status} {state.guard_age}  "
                  f"{BOLD}patches{RESET}: {GREEN}{state.n_ok} ok{RESET} "
                  f"{GREEN}{state.n_applied} applied{RESET} "
                  f"{RED if state.n_drift else DIM}{state.n_drift} drift{RESET}\n")
        else:
            print(f"  {DIM}press 1-9, 0, or q{RESET}")
