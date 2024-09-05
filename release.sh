#!/bin/bash

ver=$1

if [ "ver" = "" ]; then
    echo Release version is empty.
    exit
fi

# Windows amd64
echo release etcdkeeper-v%ver%-windows_x86_64.zip
tar -cf release\etcdkeeper-v%ver%-windows_x86_64.zip --strip-components 1 bin\etcdkeeper.exe
tar -rf release\etcdkeeper-v%ver%-windows_x86_64.zip assets
tar -rf release\etcdkeeper-v%ver%-windows_x86_64.zip LICENSE README.md

# Linux amd64
echo release etcdkeeper-v%ver%-linux_x86_64.zip
tar -cf release\etcdkeeper-v%ver%-linux_x86_64.zip --strip-components 2 bin\linux_amd64\etcdkeeper
tar -rf release\etcdkeeper-v%ver%-linux_x86_64.zip assets
tar -rf release\etcdkeeper-v%ver%-linux_x86_64.zip LICENSE README.md

# Darwin amd64
echo release etcdkeeper-v%ver%-darwin_x86_64.zip
tar -cf release\etcdkeeper-v%ver%-darwin_x86_64.zip --strip-components 2 bin\darwin_amd64\etcdkeeper
tar -rf release\etcdkeeper-v%ver%-darwin_x86_64.zip assets
tar -rf release\etcdkeeper-v%ver%-darwin_x86_64.zip LICENSE README.md

# Linux arm64
echo release etcdkeeper-v%ver%-linux_arm64.zip
tar -cf release\etcdkeeper-v%ver%-linux_arm64.zip --strip-components 2 bin\linux_arm64\etcdkeeper
tar -rf release\etcdkeeper-v%ver%-linux_arm64.zip assets
tar -rf release\etcdkeeper-v%ver%-linux_arm64.zip LICENSE README.md