#!/usr/bin/env bash

cd "$(dirname "$0")/.." || exit
set -euo pipefail

go test -v -race ./... -count=1
