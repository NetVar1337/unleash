# vpcc fix-plugin-hook-paths.ps1 -- Windows-only manifest rewriter.
#
# Problem: CC plugin manifests use ${CLAUDE_PLUGIN_ROOT} in hook command
# strings. On Windows launched from a Git Bash / MSYS context, CC resolves
# that to a POSIX path like /c/Users/foo/...; Windows Node treats it as a
# relative drive-root path and tries C:\c\Users\foo\... -> MODULE_NOT_FOUND.
# Vanilla CC denies the hook before this fires; vpcc removes the deny gates
# so the bug surfaces.
#
# This script is the on-disk half. claude-preload.js section 7 handles the
# runtime half (env normalization + child_process argv normalization).
#
# Walks every hooks.json under %USERPROFILE%\.claude\plugins\{cache,marketplaces}
# and substitutes:
#   1. ${CLAUDE_PLUGIN_ROOT}            -> <resolved plugin root, forward-slash>
#   2. %CLAUDE_PLUGIN_ROOT%             -> <resolved plugin root, forward-slash>
#   3. /X/path tokens (POSIX absolute)  -> X:/path  (token-boundary aware)
#
# Idempotent. Backs up original once per file as *.vpcc-bak.
# Upstream CC bug refs: anthropics/claude-code#24529, #16116, #25184.

[CmdletBinding()]
param(
    [string]$ClaudeRoot = (Join-Path $env:USERPROFILE '.claude'),
    [switch]$WhatIfOnly
)

$ErrorActionPreference = 'Stop'

function Write-Info($m) { Write-Host "[vpcc-fix] $m" -ForegroundColor Cyan }
function Write-Ok2($m)  { Write-Host "[  ok   ] $m" -ForegroundColor Green }
function Write-Skip2($m){ Write-Host "[ skip  ] $m" -ForegroundColor DarkGray }
function Write-Warn2($m){ Write-Host "[ warn  ] $m" -ForegroundColor Yellow }

function Repair-CommandString {
    param(
        [string]$Command,
        [string]$PluginRoot
    )
    if (-not $Command) { return $Command }

    # Use forward-slash form. Windows Node accepts forward slashes and JSON
    # encoding does not need to escape them.
    $rootForward = $PluginRoot -replace '\\', '/'

    $out = $Command
    $out = $out.Replace('${CLAUDE_PLUGIN_ROOT}', $rootForward)
    $out = $out.Replace('%CLAUDE_PLUGIN_ROOT%', $rootForward)

    # Token-boundary aware POSIX-path rewrite: only rewrite /X/path when the
    # /X/ appears at string start or after a quote / whitespace / equals.
    $out = [regex]::Replace(
        $out,
        '(?<prefix>["''\s=]|^)/(?<drive>[a-zA-Z])/(?<rest>[^"''\s]+)',
        {
            param($m)
            $p = $m.Groups['prefix'].Value
            $d = $m.Groups['drive'].Value.ToUpper()
            $r = $m.Groups['rest'].Value
            "$p$($d):/$r"
        }
    )

    return $out
}

function Test-NeedsRepair {
    param([string]$Command)
    if (-not $Command) { return $false }
    if ($Command -match '\$\{CLAUDE_PLUGIN_ROOT\}') { return $true }
    if ($Command -match '%CLAUDE_PLUGIN_ROOT%') { return $true }
    if ($Command -match '(["''\s=]|^)/([a-zA-Z])/[^"''\s]+') { return $true }
    return $false
}

function Repair-HooksJson {
    param([string]$ManifestPath)

    # Plugin root = directory two levels up from hooks/hooks.json.
    $pluginRoot = (Get-Item (Split-Path -Parent (Split-Path -Parent $ManifestPath))).FullName

    try {
        $raw = Get-Content -Raw -LiteralPath $ManifestPath -Encoding UTF8
    } catch {
        Write-Warn2 "could not read $ManifestPath ($_)"
        return $false
    }

    try {
        $obj = $raw | ConvertFrom-Json -ErrorAction Stop
    } catch {
        Write-Warn2 "invalid JSON, skipping: $ManifestPath"
        return $false
    }

    if (-not $obj.hooks) {
        Write-Skip2 "no hooks section: $ManifestPath"
        return $false
    }

    $repairedAny = $false

    foreach ($eventName in @($obj.hooks.PSObject.Properties.Name)) {
        $entries = $obj.hooks.$eventName
        if (-not $entries) { continue }
        foreach ($entry in $entries) {
            if (-not $entry.hooks) { continue }
            foreach ($h in $entry.hooks) {
                if (-not $h.command) { continue }
                if (-not (Test-NeedsRepair $h.command)) { continue }
                $fixed = Repair-CommandString -Command $h.command -PluginRoot $pluginRoot
                if ($fixed -ne $h.command) {
                    $h.command = $fixed
                    $repairedAny = $true
                }
            }
        }
    }

    if (-not $repairedAny) {
        Write-Skip2 "already clean: $ManifestPath"
        return $false
    }

    $newRaw = ($obj | ConvertTo-Json -Depth 32)

    if ($WhatIfOnly) {
        Write-Info "would rewrite: $ManifestPath"
        return $true
    }

    $backup = "$ManifestPath.vpcc-bak"
    if (-not (Test-Path $backup)) {
        Copy-Item -LiteralPath $ManifestPath -Destination $backup -Force
    }

    Set-Content -LiteralPath $ManifestPath -Value $newRaw -Encoding UTF8
    Write-Ok2 "rewrote: $ManifestPath"
    return $true
}

if (-not (Test-Path $ClaudeRoot)) {
    Write-Warn2 "Claude root not found: $ClaudeRoot, nothing to do"
    return
}

$searchRoots = @(
    Join-Path $ClaudeRoot 'plugins\cache'
    Join-Path $ClaudeRoot 'plugins\marketplaces'
) | Where-Object { Test-Path $_ }

if (-not $searchRoots) {
    Write-Info "no plugin cache or marketplaces dirs present yet, nothing to do"
    return
}

$manifests = foreach ($r in $searchRoots) {
    Get-ChildItem -Path $r -Filter 'hooks.json' -Recurse -File -ErrorAction SilentlyContinue |
        Where-Object { $_.FullName -match '\\hooks\\hooks\.json$' }
}

if (-not $manifests) {
    Write-Info "no plugin hooks.json files found under $($searchRoots -join ', ')"
    return
}

$total = ($manifests | Measure-Object).Count
$changed = 0
Write-Info "scanning $total plugin manifest(s)"
foreach ($m in $manifests) {
    if (Repair-HooksJson -ManifestPath $m.FullName) { $changed++ }
}
$clean = $total - $changed
Write-Info "done. $changed manifest(s) rewritten, $clean already clean"
