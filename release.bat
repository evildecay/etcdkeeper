@echo off

if "%OS%" == "Windows_NT" (
    setlocal enabledelayedexpansion
)

set ver=%1

if "%ver%" == "" (
    echo Release version is empty.
    goto quit
)

REM Windows amd64
echo release etcdkeeper-v%ver%-windows_x86_64.tar.gz
copy bin\etcdkeeper.exe etcdkeeper.exe
tar -czf release\etcdkeeper-v%ver%-windows_x86_64.tar.gz etcdkeeper.exe assets LICENSE README.md
del etcdkeeper.exe

REM Linux amd64
echo release etcdkeeper-v%ver%-linux_x86_64.tar.gz
copy bin\linux_amd64\etcdkeeper etcdkeeper
tar -czf release\etcdkeeper-v%ver%-linux_x86_64.tar.gz etcdkeeper assets LICENSE README.md
del etcdkeeper

REM Darwin amd64
echo release etcdkeeper-v%ver%-darwin_x86_64.tar.gz
copy bin\darwin_amd64\etcdkeeper etcdkeeper
tar -czf release\etcdkeeper-v%ver%-darwin_x86_64.tar.gz etcdkeeper assets LICENSE README.md
del etcdkeeper

REM Linux arm64
echo release etcdkeeper-v%ver%-linux_arm64.tar.gz
copy bin\linux_arm64\etcdkeeper etcdkeeper
tar -czf release\etcdkeeper-v%ver%-linux_arm64.tar.gz etcdkeeper assets LICENSE README.md
del etcdkeeper

:quit

endlocal