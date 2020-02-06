#!/usr/bin/env bash
set -e
source ./scripts/build.sh
./numbers --concurrent-connections ${1-5} --port ${2-4000}