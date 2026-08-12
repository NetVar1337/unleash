@echo off
rem Integration verification: sync embedded assets, rebuild, dry-run all targets.
setlocal
cd /d %~dp0..

echo [1/4] syncing embedded assets...
call node scripts\sync-assets.mjs || exit /b 1

echo [2/4] rebuilding unleash...
cd go
go build -o ..\dist\unleash-current.exe . || exit /b 1
cd ..

echo [3/4] dry-run patch against all detected targets...
dist\unleash-current.exe patch --dry-run > dist\dryrun.log 2>&1
type dist\dryrun.log | findstr /c:"drift (search not found" > dist\drift-lines.log
for /f %%N in ('type dist\drift-lines.log ^| find /c /v ""') do set DRIFT=%%N
for /f %%N in ('type dist\dryrun.log ^| findstr /c:"would try" ^| find /c /v ""') do set TRY=%%N
echo   drift=%DRIFT%  try=%TRY%

echo [4/4] go test...
cd go
go test ./... || exit /b 1
echo INTEGRATION_OK drift=%DRIFT% try=%TRY%
