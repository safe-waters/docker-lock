#! /usr/bin/env bash

cd "$(dirname "$0")" || exit

set -euo pipefail

docker lock generate
docker lock verify
docker lock rewrite --tempdir .

echo "------ PASSED CONTRIB TESTS ------"
