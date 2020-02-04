#!/usr/bin/env bash
set -e
cd cmd/server
go build -o numbers .
chmod a+x numbers
./numbers --concurrent-connections 5