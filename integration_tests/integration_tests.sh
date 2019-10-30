#! /usr/bin/env bash
(
    cd "$(dirname "$0")/.." || exit
    run_integration_tests() {
        # docker logged out with no creds in .env, generate should fail
        if ! docker lock verify --env-file .envwithoutcreds > /dev/null 2>&1; then
            echo -e "\n ------ PASSED: docker lock failed after docker logout ------ \n"
        else
            echo -e "\n ------ ERROR: docker lock succeeded after docker logout ------ \n"
            exit 1
        fi

        # using .env but still logged out, generate should succeed
        if docker lock verify --env-file .envwithcreds > /dev/null 2>&1; then
            echo -e "\n ------ PASSED: docker lock succeeded after docker logout with .env credentials ------ \n"
        else
            echo -e "\n ------ ERROR: docker lock failed after docker logout with .env credentials ------ \n"
            exit 1
        fi

        # docker login again, generate should succeed
        docker login $3 --username $1 --password $2 > /dev/null 2>&1
        if docker lock verify --env-file .envwithoutcreds > /dev/null 2>&1; then
            echo -e "\n ------ PASSED: docker lock succeeded after docker login again ------ \n"
        else
            echo -e "\n ------ ERROR: docker lock failed after docker login again ------ \n"
            exit 1
        fi

        docker logout $3 > /dev/null 2>&1
    }

    cd docker/
    USERNAME="$DOCKER_USERNAME"
    PASSWORD="$DOCKER_PASSWORD"
    envsubst < .envreplacewithcreds > .envwithcreds
    envsubst < .envreplacewithoutcreds > .envwithoutcreds
    unset DOCKER_USERNAME
    unset DOCKER_PASSWORD
    run_integration_tests "$USERNAME" "$PASSWORD"
    cd ..

    echo -e "\n ------ PASSED PRIVATE DOCKER TESTS ------ \n"

    cd acr/
    USERNAME="$ACR_USERNAME"
    PASSWORD="$ACR_PASSWORD"
    SERVER="$ACR_REGISTRY_NAME.azurecr.io"
    envsubst < .envreplacewithcreds > .envwithcreds
    envsubst < .envreplacewithoutcreds > .envwithoutcreds
    unset ACR_USERNAME
    unset ACR_PASSWORD
    unset ACR_REGISTRY_NAME
    run_integration_tests "$USERNAME" "$PASSWORD" "$SERVER"
    cd ..

    echo -e "\n ------ PASSED PRIVATE ACR TESTS ------ \n"
)
