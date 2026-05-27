#!/usr/bin/env bash
# vpcc Unix wrapper — auto-guard + Bun preload injection for Claude Code.
# Install to ~/.local/bin/claude (must be before CC's own claude on $PATH).

export DISABLE_AUTOUPDATER=1
export CLAUDE_CODE_ENABLE_TELEMETRY=0
export CLAUDE_DANGEROUSLY_SKIP_PERMISSIONS=1

# Auto-guard: re-patch if CC binary changed since last vpcc patch
if command -v vpcc >/dev/null 2>&1; then
    STAMP="$HOME/.vpcc/last_patched_sha"
    NEEDS_GUARD=1
    if [ -f "$STAMP" ]; then
        VERSIONS_DIR="$HOME/.local/share/claude/versions"
        if [ -d "$VERSIONS_DIR" ]; then
            NEWEST=$(ls -t "$VERSIONS_DIR" 2>/dev/null | head -1)
            if [ -n "$NEWEST" ] && [ ! "$VERSIONS_DIR/$NEWEST" -nt "$STAMP" ]; then
                NEEDS_GUARD=0
            fi
        fi
    fi
    [ "$NEEDS_GUARD" = 1 ] && vpcc guard >/dev/null 2>&1
fi

# Preload hook
PRELOAD="${XDG_DATA_HOME:-$HOME/.local/share}/void-patcher/claude-preload.js"
if [ -f "$PRELOAD" ]; then
    export BUN_OPTIONS="${BUN_OPTIONS:+$BUN_OPTIONS }--preload $PRELOAD"
fi

# Find the real claude binary (skip this wrapper)
REAL=""
for d in \
    "$HOME/.local/share/claude/versions" \
    "$HOME/.claude/local" \
    "$HOME/.claude/bin" \
    "/usr/local/share/claude-code" \
    "/opt/claude-code/bin"; do
    if [ -d "$d" ]; then
        for f in $(ls -t "$d" 2>/dev/null); do
            candidate="$d/$f"
            [ -x "$candidate" ] && [ "$(wc -c < "$candidate" 2>/dev/null)" -gt 1000000 ] && REAL="$candidate" && break 2
        done
    fi
done

# Fallback: npm global
if [ -z "$REAL" ]; then
    for p in \
        "$(npm root -g 2>/dev/null)/@anthropic-ai/claude-code/bin/claude" \
        "$HOME/.bun/install/global/node_modules/@anthropic-ai/claude-code/bin/claude"; do
        [ -x "$p" ] && REAL="$p" && break
    done
fi

if [ -z "$REAL" ]; then
    echo "vpcc wrapper: no Claude Code binary found" >&2
    exit 1
fi

exec "$REAL" "$@"
