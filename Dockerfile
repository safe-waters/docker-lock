ARG GO_VERSION=1.15.0
FROM golang:${GO_VERSION} AS build
SHELL ["/bin/bash", "-euo", "pipefail", "-c"]
WORKDIR /build
ARG IMAGE_TAG=latest
## linux/amd64, linux/arm64 supported
ARG TARGETPLATFORM
COPY . .
RUN curl -fsSL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | bash && \
     mv ./bin/goreleaser /usr/local/bin && \
     if [[ "${IMAGE_TAG}" == "latest" ]]; then goreleaser build --snapshot; else goreleaser build; fi
RUN ls -al dist
    TARGETPLATFORM="${TARGETPLATFORM/\//_}" && \
    mv "dist/docker-lock_${TARGETPLATFORM}/docker-lock" prod

FROM scratch AS prod
ARG IMAGE_TAG=latest
ARG TARGETPLATFORM
COPY --from=build /build/prod/docker-lock .
ENTRYPOINT ["./docker-lock"]
