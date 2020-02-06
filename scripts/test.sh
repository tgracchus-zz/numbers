#!/usr/bin/env bash
set -e
go clean -testcache
go test -v -cover -race  tgracchus/numbers
