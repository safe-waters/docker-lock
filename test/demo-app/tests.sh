#! /usr/bin/env bash

set -euo pipefail

function cleanup() {
    rm ./*-test.json
}

function compare_lockfiles() {
    local lockfile
    local test_lockfile

    lockfile="${1}"
    test_lockfie="${2}"

    if ! diff docker-lock.json docker-lock-test.json; then
        exit 1
    fi
}

trap cleanup EXIT

docker lock generate --lockfile-name docker-lock-test.json
docker lock verify --lockfile-name docker-lock.json
compare_lockfiles docker-lock.json docker-lock-test.json

docker lock generate --exclude-all-dockerfiles --lockfile-name docker-lock-exclude-all-dockerfiles-test.json
docker lock verify --lockfile-name docker-lock-exclude-all-dockerfiles.json
compare_lockfiles docker-lock-exclude-all-dockerfiles.json docker-lock-exclude-all-dockerfiles-test.json

docker lock generate --exclude-all-composefiles --lockfile-name docker-lock-exclude-all-composefiles-test.json
docker lock verify --lockfile-name docker-lock-exclude-all-composefiles.json
compare_lockfiles docker-lock-exclude-all-composefiles.json docker-lock-exclude-all-composefiles-test.json

docker lock generate --lockfile-name docker-lock-base-dir-test.json
docker lock verify --lockfile-name docker-lock-base-dir-test.json
compare_lockfiles docker-lock-base-dir.json docker-lock-base-dir-test.json

docker lock generate --dockerfiles web/Dockerfile --lockfile-name docker-lock-dockerfiles-test.json
docker lock verify --lockfile-name docker-lock-dockerfiles-test.json
compare_lockfiles docker-lock-dockerfiles.json docker-lock-dockerfiles-test.json

docker lock generate --composefiles docker-compose-1.yml --lockfile-name docker-lock-composefiles-test.json
docker lock verify --lockfile-name docker-lock-composefiles-test.json
compare_lockfiles docker-lock-composefiles.json docker-lock-composefiles-test.json

docker lock generate --dockerfile-recursive --lockfile-name docker-lock-dockerfile-recursive-test.json
docker lock verify --lockfile-name docker-lock-dockerfile-recursive-test.json
compare_lockfiles docker-lock-dockerfile-recursive.json docker-lock-dockerfile-recursive-test.json 

docker lock generate --composefile-recursive --lockfile-name docker-lock-composefile-recursive-test.json
docker lock verify --lockfile-name docker-lock-composefile-recursive-test.json
compare_lockfiles docker-lock-composefile-recursive.json docker-lock-composefile-recursive-test.json