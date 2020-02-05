#!/usr/bin/env bash
set -e
cd cmd/server
go build -o numbers .
chmod a+x numbers
./numbers --concurrent-connections ${1-5} --port ${2-4000}