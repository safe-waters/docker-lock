#!/usr/bin/env bash

cd "$(dirname "$0")/.." || exit
set -euo pipefail

go test -race ./... -coverprofile=coverage.out
