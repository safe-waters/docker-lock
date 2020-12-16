#! /usr/bin/env bash

cd "$(dirname "$0")" || exit

set -euo pipefail

function cleanup() {
    rm ./*-test.json
}

function diff_files() {
    local lockfile
    local test_lockfile

    lockfile="${1}"
    test_lockfile="${2}"

    if ! diff "${lockfile}" "${test_lockfile}"; then
        exit 1
    fi
}

function run_generate_verify_tests() {
    echo "------ RUNNING GENERATE/VERIFY TESTS ------"

    echo "default"
    docker lock generate --lockfile-name docker-lock-test.json
    docker lock verify --lockfile-name docker-lock.json
    diff_files docker-lock.json docker-lock-test.json

    echo "--exclude-all-dockerfiles"
    docker lock generate --exclude-all-dockerfiles --lockfile-name docker-lock-exclude-all-dockerfiles-test.json
    docker lock verify --lockfile-name docker-lock-exclude-all-dockerfiles.json
    diff_files docker-lock-exclude-all-dockerfiles.json docker-lock-exclude-all-dockerfiles-test.json

    echo "--exclude-all-composefiles"
    docker lock generate --exclude-all-composefiles --lockfile-name docker-lock-exclude-all-composefiles-test.json
    docker lock verify --lockfile-name docker-lock-exclude-all-composefiles.json
    diff_files docker-lock-exclude-all-composefiles.json docker-lock-exclude-all-composefiles-test.json

    echo "--exclude-all-kubernetesfiles"
    docker lock generate --exclude-all-kubernetesfiles --lockfile-name docker-lock-exclude-all-kubernetesfiles-test.json
    docker lock verify --lockfile-name docker-lock-exclude-all-kubernetesfiles.json
    diff_files docker-lock-exclude-all-kubernetesfiles.json docker-lock-exclude-all-kubernetesfiles-test.json

    echo "--base-dir"
    docker lock generate --base-dir web --lockfile-name docker-lock-base-dir-test.json
    docker lock verify --lockfile-name docker-lock-base-dir-test.json
    diff_files docker-lock-base-dir.json docker-lock-base-dir-test.json

    echo "--dockerfiles"
    docker lock generate --dockerfiles web/Dockerfile --lockfile-name docker-lock-dockerfiles-test.json
    docker lock verify --lockfile-name docker-lock-dockerfiles-test.json
    diff_files docker-lock-dockerfiles.json docker-lock-dockerfiles-test.json

    echo "--composefiles"
    docker lock generate --composefiles docker-compose.yml,docker-compose-1.yml --lockfile-name docker-lock-composefiles-test.json
    docker lock verify --lockfile-name docker-lock-composefiles-test.json
    diff_files docker-lock-composefiles.json docker-lock-composefiles-test.json

    echo "--kubernetesfiles"
    docker lock generate --kubernetesfiles database/pod.yaml --lockfile-name docker-lock-kubernetesfiles-test.json
    docker lock verify --lockfile-name docker-lock-kubernetesfiles-test.json
    diff_files docker-lock-kubernetesfiles.json docker-lock-kubernetesfiles-test.json

    echo "--dockerfile-recursive"
    docker lock generate --dockerfile-recursive --lockfile-name docker-lock-dockerfile-recursive-test.json
    docker lock verify --lockfile-name docker-lock-dockerfile-recursive-test.json
    diff_files docker-lock-dockerfile-recursive.json docker-lock-dockerfile-recursive-test.json

    echo "--composefile-recursive"
    docker lock generate --composefile-recursive --lockfile-name docker-lock-composefile-recursive-test.json
    docker lock verify --lockfile-name docker-lock-composefile-recursive-test.json
    diff_files docker-lock-composefile-recursive.json docker-lock-composefile-recursive-test.json

    echo "--kubernetesfile-recursive"
    docker lock generate --kubernetesfile-recursive --lockfile-name docker-lock-kubernetesfile-recursive-test.json
    docker lock verify --lockfile-name docker-lock-kubernetesfile-recursive-test.json
    diff_files docker-lock-kubernetesfile-recursive.json docker-lock-kubernetesfile-recursive-test.json

    echo "--dockerfile-globs"
    docker lock generate --dockerfile-globs 'web/Docker*','database/Docker*' --lockfile-name docker-lock-dockerfile-globs-test.json
    docker lock verify --lockfile-name docker-lock-dockerfile-globs-test.json
    diff_files docker-lock-dockerfile-globs.json docker-lock-dockerfile-globs-test.json

    echo "--composefile-globs"
    docker lock generate --composefile-globs 'docker-compose*.yml' --lockfile-name docker-lock-composefile-globs-test.json
    docker lock verify --lockfile-name docker-lock-composefile-globs-test.json
    diff_files docker-lock-composefile-globs.json docker-lock-composefile-globs-test.json

    echo "--kubernetesfile-globs"
    docker lock generate --kubernetesfile-globs 'database/po*' --lockfile-name docker-lock-kubernetesfile-globs-test.json
    docker lock verify --lockfile-name docker-lock-kubernetesfile-globs-test.json
    diff_files docker-lock-kubernetesfile-globs.json docker-lock-kubernetesfile-globs-test.json

    echo "--ignore-missing-digests"
    docker lock generate --ignore-missing-digests --dockerfiles "private/Dockerfile-errors" --lockfile-name docker-lock-ignore-missing-digests-test.json
    docker lock verify --ignore-missing-digests --lockfile-name docker-lock-ignore-missing-digests-test.json
    diff_files docker-lock-ignore-missing-digests.json docker-lock-ignore-missing-digests-test.json
    echo "------ PASSED GENERATE/VERIFY TESTS ------"
}

function run_rewrite_verify_tests() {
    echo "------ RUNNING REWRITE/VERIFY TESTS ------"

    echo "default"
    docker lock rewrite
    docker lock verify
    diff_files docker-compose.yml docker-compose-rewrite.yml
    diff_files web/Dockerfile web/Dockerfile-rewrite
    diff_files database/Dockerfile database/Dockerfile-rewrite
    diff_files pod.yml pod-rewrite.yml

    echo "--exclude-tags"
    docker lock rewrite --exclude-tags
    docker lock verify --exclude-tags
    diff_files docker-compose.yml docker-compose-rewrite-exclude-tags.yml
    diff_files web/Dockerfile web/Dockerfile-rewrite-exclude-tags
    diff_files database/Dockerfile database/Dockerfile-rewrite-exclude-tags
    diff_files pod.yml pod-rewrite-exclude-tags.yml

    echo "------ PASSED REWRITE/VERIFY TESTS ------"
}

function main() {
    trap cleanup EXIT

    docker login --username "${DOCKER_USERNAME}" --password "${DOCKER_PASSWORD}" > /dev/null 2>&1

    run_generate_verify_tests
    run_rewrite_verify_tests
}

main
