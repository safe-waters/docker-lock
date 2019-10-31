#! /usr/bin/env bash
(
    cd "$(dirname "$0")" || exit
    integration_tests_dir="$(pwd)"

    set -euo pipefail
    IFS=$'\n\t'

    cleanup () {
        cd "${integration_tests_dir}"
        rm ./**/.envwithcreds
        rm ./**/.envwithoutcreds
    }

    before_test () {
        envsubst < .envreplacewithcreds > .envwithcreds
        envsubst < .envreplacewithoutcreds > .envwithoutcreds
        # shellcheck disable=SC2046
        unset $(cut -d= -f1 .envreplacewithcreds)
    }

    after_test () {
        cd "${integration_tests_dir}"
    }

    run_integration_tests() {
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

    main() {
        trap cleanup EXIT

        cd docker/
        USERNAME="${DOCKER_USERNAME}"
        PASSWORD="${DOCKER_PASSWORD}"
        before_test
        run_integration_tests "${USERNAME}" "${PASSWORD}" ""
        after_test

        echo "------ PASSED PRIVATE DOCKER TESTS ------"

        cd acr/
        USERNAME="${ACR_USERNAME}"
        PASSWORD="${ACR_PASSWORD}"
        SERVER="${ACR_REGISTRY_NAME}.azurecr.io"
        before_test
        run_integration_tests "${USERNAME}" "${PASSWORD}" "${SERVER}"
        after_test

        echo "------ PASSED PRIVATE ACR TESTS ------"
    }

    main
)
