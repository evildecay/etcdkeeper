@echo off

if "%OS%" == "Windows_NT" (
    setlocal enabledelayedexpansion
)

cd src\etcdkeeper

REM Windows amd64
set GOOS=windows
set GOARCH=amd64
go install
echo build etcdkeeper GOOS=windows GOARCH=amd64 ok

REM Linux amd64
set GOOS=linux
set GOARCH=amd64
go install
echo build etcdkeeper GOOS=linux GOARCH=amd64 ok

REM Darwin amd64
set GOOS=darwin
set GOARCH=amd64
go install
echo build etcdkeeper GOOS=darwin GOARCH=amd64 ok

REM Linux arm64
set GOOS=linux
set GOARCH=arm64
go install
echo build etcdkeeper GOOS=linux GOARCH=arm64 ok

cd ..\..

endlocal