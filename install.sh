#!/usr/bin/env bash
# vpcc 1-liner installer — macOS · Linux · WSL
#
#   curl -fsSL https://raw.githubusercontent.com/VoidChecksum/void-patcher-cc/main/install.sh | bash
#
# Chains on top of Anthropic's official native installer
# (https://claude.ai/install.sh) → then deploys vpcc patches + preload hook.
# No hardcoded absolute paths — every path is $HOME/XDG derived.

set -euo pipefail

REPO="VoidChecksum/void-patcher-cc"
ANTHROPIC_INSTALL="https://claude.ai/install.sh"

BLUE='\033[1;34m'; GREEN='\033[1;32m'; YELLOW='\033[1;33m'; RED='\033[1;31m'; NC='\033[0m'
log() { printf "${BLUE}[vpcc]${NC} %s\n" "$*"; }
ok()  { printf "${GREEN}[ ok ]${NC} %s\n" "$*"; }
warn(){ printf "${YELLOW}[warn]${NC} %s\n" "$*"; }
die() { printf "${RED}[fail]${NC} %s\n" "$*" >&2; exit 1; }

XDG_BIN="${XDG_BIN_HOME:-$HOME/.local/bin}"
XDG_DATA="${XDG_DATA_HOME:-$HOME/.local/share}"
XDG_CFG="${XDG_CONFIG_HOME:-$HOME/.config}"
export PATH="$XDG_BIN:$PATH"

# ── 1. Claude Code — install only if genuinely missing ──────────────────────
# `command -v claude` may resolve to vpcc wrapper; probe the real binary.
have_real_cc() {
    command -v claude >/dev/null 2>&1 || return 1
    # accept if the resolved path runs and prints a version line
    claude --version 2>/dev/null | grep -qE 'Claude Code|claude-code' && return 0
    # fallback: file size check on known native install paths
    for p in "$HOME/.local/bin/claude" \
             "$HOME/.claude/local/claude" "$HOME/.claude/bin/claude" \
             "/opt/claude-code/bin/claude" "/usr/local/bin/claude"; do
        [ -x "$p" ] && [ "$(stat -c%s "$p" 2>/dev/null || stat -f%z "$p" 2>/dev/null || echo 0)" -gt 1000000 ] && return 0
    done
    # also check ~/.local/share/claude/versions/ (current native installer layout)
    if [ -d "$HOME/.local/share/claude/versions" ]; then
        for p in "$HOME/.local/share/claude/versions/"*; do
            [ -x "$p" ] && [ "$(stat -c%s "$p" 2>/dev/null || stat -f%z "$p" 2>/dev/null || echo 0)" -gt 1000000 ] && return 0
        done
    fi
    return 1
}
if have_real_cc; then
    ok "claude already installed — skipping Anthropic installer"
else
    log "installing Claude Code via $ANTHROPIC_INSTALL"
    curl -fsSL "$ANTHROPIC_INSTALL" | bash || die "Anthropic installer failed"
fi

# ── 2. Python + pipx (skip if present) ───────────────────────────────────────
# Python: try python3, then python, then error with distro-specific advice
PY=""
for cand in python3 python; do
    if command -v "$cand" >/dev/null 2>&1; then PY="$cand"; break; fi
done
if [ -z "$PY" ]; then
    die "Python missing — install with: brew install python3 | sudo apt install python3 | sudo pacman -S python | sudo dnf install python3"
fi
# Note: on Arch Linux, pipx is available as the 'python-pipx' package.
if command -v pipx >/dev/null; then
    ok "pipx already installed"
else
    log "installing pipx"
    "$PY" -m pip install --user pipx >/dev/null 2>&1 || die "pipx install failed"
    "$PY" -m pipx ensurepath >/dev/null 2>&1 || true
fi

# ── 3. vpcc package — install or upgrade only if needed ──────────────────────
# Compare installed version against pyproject.toml version at remote HEAD.
# pipx list output has leading whitespace: "   package vpcc 2.1.x, ..."
# so grep without ^ anchor; dots in version are string-compared, not regex.
REMOTE_VER="$(curl -fsSL "https://raw.githubusercontent.com/$REPO/main/pyproject.toml" 2>/dev/null \
    | awk -F'"' '/^version/{print $2; exit}')"
INSTALLED_VER="$(pipx list --short 2>/dev/null | awk '$1=="vpcc"{print $2}')"
if pipx list 2>/dev/null | grep -q 'package vpcc '; then
    if [ -n "$REMOTE_VER" ] && [ "$INSTALLED_VER" = "$REMOTE_VER" ]; then
        ok "vpcc $INSTALLED_VER already current — skipping reinstall"
    else
        log "upgrading vpcc ${INSTALLED_VER:-(unknown)} → ${REMOTE_VER:-(unknown)}"
        pipx install --force "git+https://github.com/$REPO" >/dev/null || die "pipx upgrade failed"
    fi
else
    log "installing vpcc from git+$REPO"
    pipx install "git+https://github.com/$REPO" >/dev/null || die "pipx install vpcc failed"
fi

# ── 4. Clone repo locally for contrib/ (preload, systemd units) ──────────────
REPO_DIR="$XDG_DATA/void-patcher-cc"
if [ ! -d "$REPO_DIR/.git" ]; then
    log "cloning $REPO to $REPO_DIR"
    git clone --depth=1 "https://github.com/$REPO" "$REPO_DIR" >/dev/null 2>&1 || \
        warn "clone failed — preload / systemd steps skipped"
else
    (cd "$REPO_DIR" && git pull --ff-only >/dev/null 2>&1) || true
fi

# ── 5. Apply patches + preload hook ──────────────────────────────────────────
log "applying signature patches"
vpcc patch || warn "patch returned non-zero — run 'vpcc doctor'"

log "installing runtime preload hook"
vpcc install-preload || warn "preload install failed"

# ── 6. Optional systemd --user autoheal timer (Linux only) ───────────────────
if command -v systemctl >/dev/null 2>&1 && [ -n "${XDG_RUNTIME_DIR:-}" ] && [ -d "$REPO_DIR/contrib/systemd" ]; then
    UNIT_DIR="$XDG_CFG/systemd/user"
    mkdir -p "$UNIT_DIR"
    cp -f "$REPO_DIR/contrib/systemd/"*.service "$UNIT_DIR/" 2>/dev/null || true
    cp -f "$REPO_DIR/contrib/systemd/"*.timer   "$UNIT_DIR/" 2>/dev/null || true
    systemctl --user daemon-reload 2>/dev/null || true
    if systemctl --user enable --now vpcc-autoheal.timer 2>/dev/null; then
        ok "autoheal timer active — runs every 15 min"
    fi
fi

# ── 7. Verify ────────────────────────────────────────────────────────────────
printf "\n"
vpcc doctor || true
printf "\n"
ok "install complete"
ok "usage:  vpcc patch · vpcc scan · vpcc watch · vpcc doctor"
