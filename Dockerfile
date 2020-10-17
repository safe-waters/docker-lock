ARG GO_VERSION=1.15.0
FROM golang:${GO_VERSION} AS build
SHELL ["/bin/bash", "-euo", "pipefail", "-c"]
WORKDIR /build
ARG DOCKER_LOCK_IMAGE_TAG=latest
ARG TARGETPLATFORM
COPY . .
RUN curl -fsSL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | bash && \
    mv ./bin/goreleaser /usr/local/bin && \
    if [[ "${DOCKER_LOCK_IMAGE_TAG}" == "latest" ]]; then goreleaser build --snapshot; else goreleaser build; fi && \
    TARGETPLATFORM="${TARGETPLATFORM/\//_}" && \
    mkdir prod && \
    mv "dist/docker-lock_${TARGETPLATFORM}/docker-lock" prod/

FROM scratch AS prod
ARG TARGETPLATFORM
COPY --from=build /build/prod/docker-lock .
ENTRYPOINT ["./docker-lock"]
