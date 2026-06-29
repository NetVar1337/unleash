@echo off
REM Copy patches and contrib into go\embed\ so go:embed can access them.
REM Run from repo root: go\copy-assets.bat

set SCRIPT_DIR=%~dp0
pushd %SCRIPT_DIR%\..
set REPO_ROOT=%CD%
popd

if not exist "%SCRIPT_DIR%embed\patches" mkdir "%SCRIPT_DIR%embed\patches"
if not exist "%SCRIPT_DIR%embed\contrib" mkdir "%SCRIPT_DIR%embed\contrib"

xcopy /Y /Q "%REPO_ROOT%\patches\*.json" "%SCRIPT_DIR%embed\patches\"
xcopy /Y /Q /E /I "%REPO_ROOT%\contrib" "%SCRIPT_DIR%embed\contrib\"

echo Assets copied to go\embed\
