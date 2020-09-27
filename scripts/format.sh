#!/usr/bin/env bash

(
    cd "$(dirname "$0")/.." || exit
    gofmt -s -w .
)
