# unleash installer (Windows)
# Usage: irm https://raw.githubusercontent.com/NetVar1337/unleash/main/scripts/install.ps1 | iex
#
# Downloads the latest unleash / unleash-gpt / unleash-omp release binaries
# from GitHub into %USERPROFILE%\.local\bin (create it and add it to PATH
# if missing). Existing binaries are overwritten.

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$repo = "NetVar1337/unleash"
$installDir = Join-Path $env:USERPROFILE ".local\bin"
New-Item -ItemType Directory -Force -Path $installDir | Out-Null

function Get-LatestTag([string]$prefix) {
    $releases = Invoke-RestMethod "https://api.github.com/repos/$repo/releases?per_page=30"
    $best = $null
    foreach ($r in $releases) {
        foreach ($t in $r.tag_name) {
            if ($t -like "$prefix*") {
                if ($null -eq $best) { $best = $r }
            }
        }
        if ($r.tag_name -like "$prefix*") {
            if ($null -eq $best) { $best = $r }
        }
    }
    return $best
}

function Install-Binary([string]$prefix, [string]$assetPattern, [string]$destName) {
    $release = Get-LatestTag $prefix
    if (-not $release) {
        Write-Warning "no release found for tag prefix '$prefix' — skipped $destName"
        return
    }
    $asset = $release.assets | Where-Object { $_.name -like $assetPattern } | Select-Object -First 1
    if (-not $asset) {
        Write-Warning "release $($release.tag_name) has no asset matching '$assetPattern' — skipped $destName"
        return
    }
    $dest = Join-Path $installDir $destName
    Write-Host "  downloading $($asset.name) ($($release.tag_name)) -> $dest"
    Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $dest
}

Write-Host "unleash installer"
Write-Host "  install dir: $installDir"

Install-Binary "cc-v"  "unleash-windows-amd64.exe"     "unleash.exe"
Install-Binary "gpt-v" "unleash-gpt-windows-amd64.exe" "unleash-gpt.exe"
Install-Binary "omp-v" "unleash-omp-windows-amd64.exe" "unleash-omp.exe"

# Ensure install dir is on the user PATH
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (($userPath -split ";") -notcontains $installDir) {
    [Environment]::SetEnvironmentVariable("Path", "$installDir;$userPath", "User")
    Write-Host "  added $installDir to user PATH (restart your shell)"
}

Write-Host ""
Write-Host "Done. Next:"
Write-Host "  unleash setup        # Claude Code"
Write-Host "  unleash-gpt setup    # Codex CLI"
Write-Host "  unleash-omp setup    # Oh-My-Pi"
