#!/bin/bash

cd src/etcdkeeper || echo Need to cd to the src/etcdkeeper directory first.

# Windows amd64
export GOOS=windows
export GOARCH=amd64
go install
echo build etcdkeeper GOOS=windows GOARCH=amd64 ok

# Linux amd64
export GOOS=linux
export GOARCH=amd64
go install
echo build etcdkeeper GOOS=linux GOARCH=amd64 ok

# Darwin amd64
export GOOS=darwin
export GOARCH=amd64
go install
echo build etcdkeeper GOOS=darwin GOARCH=amd64 ok

# Linux arm64
export GOOS=linux
export GOARCH=arm64
go install
echo build etcdkeeper GOOS=linux GOARCH=arm64 ok

cd ../..
