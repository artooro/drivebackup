#!/bin/bash

echo "Building darwin.amd64"
export GOOS="darwin"
export GOARCH="amd64"
go build -v -o pkg/drivebackup.darwin-amd64/drivebackup
tar -czvf pkg/drivebackup.darwin-amd64.tar.gz pkg/drivebackup.darwin-amd64

echo "Building linux.amd64"
export GOOS="linux"
export GOARCH="amd64"
go build -v -o pkg/drivebackup.linux-amd64/drivebackup
tar -czvf pkg/drivebackup.linux-amd64.tar.gz pkg/drivebackup.linux-amd64

echo "Building linux.386"
export GOOS="linux"
export GOARCH="386"
go build -v -o pkg/drivebackup.linux-386/drivebackup
tar -czvf pkg/drivebackup.linux-386.tar.gz pkg/drivebackup.linux-386

echo "Building linux.armv7"
export GOARM=7
export GOARCH=arm
go build -v -o pkg/drivebackup.linux-armv7/drivebackup
tar -czvf pkg/drivebackup.linux-armv7.tar.gz pkg/drivebackup.linux-armv7

unset GOOS
unset GOARCH
echo "Cross build complete"