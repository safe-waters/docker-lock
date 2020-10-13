#!/usr/bin/env bash

cd "$(dirname "$0")/.." || exit
set -euo pipefail

gofmt -s -w .
