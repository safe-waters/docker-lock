#! /usr/bin/env bash

run_integration_tests() { 
    # docker logged out with no creds in .env, generate should fail
    if ! docker lock verify --env-file .envwithoutcreds; then
        echo "\n ------ PASSED: docker lock failed after docker logout ------ \n"
    else
        echo "\n ------ ERROR: docker lock succeeded after docker logout ------ \n"
        exit 1
    fi

    # using .env but still logged out, generate should succeed
    if docker lock verify --env-file .envwithcreds; then
        echo "\n ------ PASSED: docker lock succeeded after docker logout with .env credentials ------ \n"
    else 
        echo "\n ------ ERROR: docker lock failed after docker logout with .env credentials ------ \n"
        exit 1
    fi

    # docker login again, generate should succeed
    docker login $3 --username $1 --password $2
    if docker lock verify --env-file .envwithoutcreds; then
        echo "\n ------ PASSED: docker lock succeeded after docker login again ------ \n"
    else 
        echo "\n ------ ERROR: docker lock failed after docker login again ------ \n"
        exit 1
    fi

    docker logout $3
}

cd docker/
USERNAME="$DOCKER_USERNAME"
PASSWORD="$DOCKER_PASSWORD"
envsubst < .envreplacewithcreds > .envwithcreds
envsubst < .envreplacewithoutcreds > .envwithoutcreds
run_integration_tests "$USERNAME" "$PASSWORD"
cd ..

echo "\n ------ PASSED PRIVATE DOCKER TESTS ------ \n"

cd acr/
USERNAME="$ACR_USERNAME"
PASSWORD="$ACR_PASSWORD"
SERVER="$ACR_REGISTRY_NAME.azurecr.io"
envsubst < .envreplacewithcreds > .envwithcreds
envsubst < .envreplacewithoutcreds > .envwithoutcreds
run_integration_tests "$USERNAME" "$PASSWORD" "$SERVER"
cd ..

echo "\n ------ PASSED PRIVATE ACR TESTS ------ \n"

