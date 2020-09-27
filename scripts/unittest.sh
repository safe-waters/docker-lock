#!/usr/bin/env bash

(
    cd "$(dirname "$0")/.." || exit
    go test -v -race ./... -count=1
)
