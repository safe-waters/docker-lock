FROM ubuntu:bionic
SHELL ["/bin/bash", "-o", "pipefail", "-c"]
ARG GO_VERSION=1.16.5
ARG GOLANGCI_LINT_VERSION=1.27.0
RUN apt-get update && \
    apt-get install curl gcc git vim tree tmux shellcheck make gettext-base -y && \
    curl -fsSL https://get.docker.com | bash && \
    DOCKER_COMPOSE_VERSION="$(git ls-remote https://github.com/docker/compose | grep refs/tags | grep -oE "[0-9]+\.[0-9][0-9]+\.[0-9]+$" | sort --version-sort | tail -n 1)" && \
    curl -fSsL "https://github.com/docker/compose/releases/download/${DOCKER_COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose && \
    chmod +x /usr/local/bin/docker-compose && \
    curl -fSsL "https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz" | tar -xz -C /usr/local && \
    curl -fSsL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | bash -s -- -b /usr/local/bin v"${GOLANGCI_LINT_VERSION}" && \
    rm -rf /var/lib/apt/lists/*
ENV PATH="${PATH}:/usr/local/go/bin"
WORKDIR /workspaces/docker-lock
