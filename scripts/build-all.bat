@echo off
rem Build all release binaries for the three products into dist\artifacts
setlocal
cd /d %~dp0..\go
set CGO_ENABLED=0
set OUT=..\dist\artifacts
mkdir %OUT% 2>nul

call :build linux amd64 linux-amd64 ""
call :build linux arm64 linux-arm64 ""
call :build darwin amd64 darwin-amd64 ""
call :build darwin arm64 darwin-arm64 ""
call :build windows amd64 windows-amd64 .exe
call :build windows arm64 windows-arm64 .exe
echo DONE
exit /b 0

:build
set GOOS=%~1
set GOARCH=%~2
set SFX=%~3
set EXT=%~4
echo building %GOOS%/%GOARCH% ...
go build -ldflags="-s -w" -o %OUT%\unleash-%SFX%%EXT% . || exit /b 1
go build -ldflags="-s -w" -o %OUT%\unleash-gpt-%SFX%%EXT% .\cmd\unleash-gpt || exit /b 1
go build -ldflags="-s -w" -o %OUT%\unleash-omp-%SFX%%EXT% .\cmd\unleash-omp || exit /b 1
exit /b 0
