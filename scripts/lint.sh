#!/usr/bin/env bash

(
    cd "$(dirname "$0")/.." || exit
    set -euo pipefail
    find . -name '*.sh' -print0 | xargs -n1 -0 shellcheck
    golangci-lint run
)