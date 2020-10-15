#! /usr/bin/env bash

cd "$(dirname "$0")" || exit

set -euo pipefail

docker lock generate
docker lock verify
docker lock rewrite

echo "------ PASSED CONTRIB TESTS ------"
