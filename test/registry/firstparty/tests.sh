#! /usr/bin/env bash

cd "$(dirname "$0")" || exit
tests_dir="$(pwd)"

set -euo pipefail
IFS=$'\n\t'

cleanup () {
    cd "${tests_dir}"
    rm ./**/.envwithcreds
    rm ./**/.envwithoutcreds
}

before_test () {
    envsubst < .envreplacewithcreds > .envwithcreds
    envsubst < .envreplacewithoutcreds > .envwithoutcreds
    # shellcheck disable=SC2046
    unset $(cut -d= -f1 .envreplacewithcreds)
}

run_external_tests() {
    docker logout > /dev/null 2>&1
    # docker logged out with no creds in .env, generate should fail
    if ! docker lock verify --env-file .envwithoutcreds > /dev/null 2>&1; then
        echo "------ PASSED: docker lock failed after docker logout ------"
    else
        echo "------ ERROR: docker lock succeeded after docker logout ------"
        exit 1
    fi

    # using .env but still logged out, generate should succeed
    if docker lock verify --env-file .envwithcreds > /dev/null 2>&1; then
        echo "------ PASSED: docker lock succeeded after docker logout with .env credentials ------"
    else
        echo "------ ERROR: docker lock failed after docker logout with .env credentials ------"
        exit 1
    fi

    # docker login again, generate should succeed
    docker login --username "$1" --password "$2" "$3" > /dev/null 2>&1
    if docker lock verify --env-file .envwithoutcreds > /dev/null 2>&1; then
        echo "------ PASSED: docker lock succeeded after docker login again ------"
    else
        echo "------ ERROR: docker lock failed after docker login again ------"
        exit 1
    fi

    docker logout "$3" > /dev/null 2>&1
}

run_internal_tests() {
    docker logout > /dev/null 2>&1

    docker login --username "$1" --password "$2" "$3" > /dev/null 2>&1

    docker run -d -p 5000:5000 --restart=always --name registry registry:2

    docker pull dockerlocktestaccount/busybox

    docker tag dockerlocktestaccount/busybox localhost:5000/busybox

    docker push localhost:5000/busybox

    docker lock verify --env-file .envwithcreds > /dev/null 2>&1
}

main() {
    trap cleanup EXIT

    (
        cd docker/
        USERNAME="${DOCKER_USERNAME}"
        PASSWORD="${DOCKER_PASSWORD}"
        before_test
        run_external_tests "${USERNAME}" "${PASSWORD}" ""
        echo "------ PASSED PRIVATE DOCKER TESTS ------"
    )

    (
        cd acr/
        USERNAME="${ACR_USERNAME}"
        PASSWORD="${ACR_PASSWORD}"
        SERVER="${ACR_REGISTRY_NAME}.azurecr.io"
        before_test
        run_external_tests "${USERNAME}" "${PASSWORD}" "${SERVER}"
        echo "------ PASSED PRIVATE ACR TESTS ------"
    )

    (
        # only linux build agent has docker daemon
        if [[ "${1}" == "linux" ]]; then
            cd internal/
            USERNAME="${DOCKER_USERNAME}"
            PASSWORD="${DOCKER_PASSWORD}"
            before_test
            run_internal_tests "${USERNAME}" "${PASSWORD}" ""
            echo "------ PASSED INTERNAL REGISTRY TESTS  ------"
        fi
    )
}

main "${1}"
