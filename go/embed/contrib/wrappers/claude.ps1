# vpcc PowerShell wrapper — Windows Bun preload injection for Claude Code.
# Save as $env:USERPROFILE\.local\bin\claude.ps1 and put on $env:PATH.

$env:DISABLE_AUTOUPDATER                 = '1'
$env:CLAUDE_CODE_ENABLE_TELEMETRY        = '0'
$env:CLAUDE_DANGEROUSLY_SKIP_PERMISSIONS = '1'

# Auto-guard: re-patch if CC binary changed since last vpcc patch
if (Get-Command vpcc -ErrorAction SilentlyContinue) {
    $guardStamp = Join-Path $env:USERPROFILE '.vpcc\last_patched_sha'
    $needsGuard = $true
    if (Test-Path $guardStamp) {
        # Quick check: compare stamp vs newest binary mtime
        $stampTime = (Get-Item $guardStamp).LastWriteTime
        $versionsDir = Join-Path $env:USERPROFILE '.local\share\claude\versions'
        if (Test-Path $versionsDir) {
            $newest = Get-ChildItem $versionsDir -File | Sort-Object LastWriteTime -Descending | Select-Object -First 1
            if ($newest -and $newest.LastWriteTime -le $stampTime) {
                $needsGuard = $false
            }
        }
    }
    if ($needsGuard) {
        vpcc guard 2>$null | Out-Null
    }
}

$preload = Join-Path $env:LOCALAPPDATA 'void-patcher\claude-preload.js'
if (Test-Path $preload) {
    if ($env:BUN_OPTIONS) {
        $env:BUN_OPTIONS = "$env:BUN_OPTIONS --preload $preload"
    } else {
        $env:BUN_OPTIONS = "--preload $preload"
    }
}

# --- Build candidate list: native installer (newest version first), scoop, winget, npm ---
$candidates = [System.Collections.Generic.List[string]]::new()

# Native installer: versioned directories under ~/.local/share/claude/versions/
$nativeVersionsDir = Join-Path $env:USERPROFILE '.local\share\claude\versions'
if (Test-Path $nativeVersionsDir) {
    Get-ChildItem -Path $nativeVersionsDir -Directory |
        Sort-Object Name -Descending |
        ForEach-Object {
            $exe = Join-Path $_.FullName 'claude.exe'
            if (Test-Path $exe) { $candidates.Add($exe) }
        }
}

# Native installer: versioned directories under LOCALAPPDATA\Programs\claude\versions\
$localVersionsDir = Join-Path $env:LOCALAPPDATA 'Programs\claude\versions'
if (Test-Path $localVersionsDir) {
    Get-ChildItem -Path $localVersionsDir -Directory |
        Sort-Object Name -Descending |
        ForEach-Object {
            $exe = Join-Path $_.FullName 'claude.exe'
            if (Test-Path $exe) { $candidates.Add($exe) }
        }
}

# Native installer: flat LOCALAPPDATA path
$candidates.Add((Join-Path $env:LOCALAPPDATA 'Programs\claude-code\claude.exe'))

# Scoop
$candidates.Add((Join-Path $env:USERPROFILE 'scoop\apps\claude-code\current\claude.exe'))

# WinGet
$wingetBase = Join-Path $env:LOCALAPPDATA 'Microsoft\WinGet\Packages'
if (Test-Path $wingetBase) {
    Get-ChildItem -Path $wingetBase -Directory -Filter '*claude*' |
        ForEach-Object {
            $exe = Join-Path $_.FullName 'claude.exe'
            if (Test-Path $exe) { $candidates.Add($exe) }
        }
}

# npm global (fallback)
$candidates.Add("$env:APPDATA\npm\node_modules\@anthropic-ai\claude-code\node_modules\@anthropic-ai\claude-code-win32-x64\claude.exe")
$candidates.Add("$env:APPDATA\npm\node_modules\@anthropic-ai\claude-code\node_modules\@anthropic-ai\claude-code-win32-arm64\claude.exe")
$candidates.Add("$env:APPDATA\npm\node_modules\@anthropic-ai\claude-code\bin\claude.exe")

$exe = $candidates | Where-Object { Test-Path $_ } | Select-Object -First 1
if (-not $exe) {
    Write-Error "vpcc wrapper: no Claude Code binary found — checked native installer, scoop, winget, and npm paths"
    exit 1
}
& $exe @args
