@echo off
rem vpcc Windows wrapper — injects Bun preload hook + env flags before claude.exe.
rem Mirrors the Linux/macOS bash wrapper at ~/.local/bin/claude.
rem Drop on %PATH% (e.g. %USERPROFILE%\.local\bin\) to shadow the npm shim.

set "DISABLE_AUTOUPDATER=1"
set "CLAUDE_CODE_ENABLE_TELEMETRY=0"
set "CLAUDE_DANGEROUSLY_SKIP_PERMISSIONS=1"

rem Auto-guard: re-patch if CC binary changed since last vpcc patch
where vpcc >nul 2>&1 && vpcc guard >nul 2>&1

set "VPCC_PRELOAD=%LOCALAPPDATA%\void-patcher\claude-preload.js"
if exist "%VPCC_PRELOAD%" (
    if defined BUN_OPTIONS (
        set "BUN_OPTIONS=%BUN_OPTIONS% --preload %VPCC_PRELOAD%"
    ) else (
        set "BUN_OPTIONS=--preload %VPCC_PRELOAD%"
    )
)

set "CLAUDE_EXE="

rem --- Native installer: versioned directory (newest first) ---
if exist "%USERPROFILE%\.local\share\claude\versions\" (
    for /f "tokens=*" %%V in ('dir /b /o-n "%USERPROFILE%\.local\share\claude\versions\" 2^>nul') do (
        if exist "%USERPROFILE%\.local\share\claude\versions\%%V\claude.exe" (
            set "CLAUDE_EXE=%USERPROFILE%\.local\share\claude\versions\%%V\claude.exe"
            goto :found
        )
    )
)

rem --- Native installer: LOCALAPPDATA Programs versioned ---
if exist "%LOCALAPPDATA%\Programs\claude\versions\" (
    for /f "tokens=*" %%V in ('dir /b /o-n "%LOCALAPPDATA%\Programs\claude\versions\" 2^>nul') do (
        if exist "%LOCALAPPDATA%\Programs\claude\versions\%%V\claude.exe" (
            set "CLAUDE_EXE=%LOCALAPPDATA%\Programs\claude\versions\%%V\claude.exe"
            goto :found
        )
    )
)

rem --- Native installer: LOCALAPPDATA Programs flat ---
if exist "%LOCALAPPDATA%\Programs\claude-code\claude.exe" (
    set "CLAUDE_EXE=%LOCALAPPDATA%\Programs\claude-code\claude.exe"
    goto :found
)

rem --- Scoop ---
if exist "%USERPROFILE%\scoop\apps\claude-code\current\claude.exe" (
    set "CLAUDE_EXE=%USERPROFILE%\scoop\apps\claude-code\current\claude.exe"
    goto :found
)

rem --- npm global (fallback) ---
for %%I in (
    "%APPDATA%\npm\node_modules\@anthropic-ai\claude-code\node_modules\@anthropic-ai\claude-code-win32-x64\claude.exe"
    "%APPDATA%\npm\node_modules\@anthropic-ai\claude-code\node_modules\@anthropic-ai\claude-code-win32-arm64\claude.exe"
    "%APPDATA%\npm\node_modules\@anthropic-ai\claude-code\bin\claude.exe"
) do if exist "%%~I" (
    set "CLAUDE_EXE=%%~I"
    goto :found
)

:found
if "%CLAUDE_EXE%"=="" (
    echo vpcc wrapper: no Claude Code binary found — checked native installer, scoop, and npm paths 1>&2
    exit /b 1
)
"%CLAUDE_EXE%" %*
