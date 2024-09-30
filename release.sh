#!/bin/bash

ver=$1

if [ "ver" = "" ]; then
    echo Release version is empty.
    exit
fi

# Windows amd64
echo release etcdkeeper-v%ver%-windows_x86_64.tar.gz
copy bin/windows_amd64/etcdkeeper.exe etcdkeeper.exe
tar -czf release/etcdkeeper-v%ver%-windows_x86_64.tar.gz etcdkeeper.exe assets LICENSE README.md
rm -rf etcdkeeper.exe

# Linux amd64
echo release etcdkeeper-v%ver%-linux_x86_64.tar.gz
copy bin/etcdkeeper etcdkeeper
tar -czf release/etcdkeeper-v%ver%-linux_x86_64.tar.gz etcdkeeper assets LICENSE README.md
rm -rf etcdkeeper

# Darwin amd64
echo release etcdkeeper-v%ver%-darwin_x86_64.tar.gz
copy bin/darwin_amd64/etcdkeeper etcdkeeper
tar -czf release/etcdkeeper-v%ver%-darwin_x86_64.tar.gz etcdkeeper assets LICENSE README.md
rm -rf etcdkeeper

# Linux arm64
echo release etcdkeeper-v%ver%-linux_arm64.tar.gz
copy bin/linux_arm64/etcdkeeper etcdkeeper
tar -czf release/etcdkeeper-v%ver%-linux_arm64.tar.gz etcdkeeper assets LICENSE README.md
rm -rf etcdkeeper