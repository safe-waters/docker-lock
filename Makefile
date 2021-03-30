SHELL=/bin/bash -euo pipefail

.PHONY: all
all: clean format lint install lock unittest

.PHONY: format
format:
	@echo "running format target..."
	@echo "running gofmt..."
	@gofmt -s -w -l .
	@echo "gofmt passed!"
	@echo "format target passed!"

.PHONY: lint
lint:
	@echo "running lint target..."
	@echo "running shellcheck..."
	@find . -name '*.sh' -print0 | xargs -n1 -0 shellcheck
	@echo "shellcheck passed!"
	@echo "running gofmt (without persisting modifications)..."
	@[[ $$(gofmt -s -l . | wc -c) -eq 0 ]];
	@echo "gofmt passed!"
	@echo "running golangci-lint..."
	@golangci-lint run --timeout 5m0s
	@echo "golangci-lint passed!"
	@echo "lint target passed!"

.PHONY: install
install:
	@echo "running install target..."
	@echo "installing docker-lock into docker's cli-plugins folder..."
	@mkdir -p "$${HOME}/.docker/cli-plugins"
	@CGO_ENABLED=0 go build -o "$${HOME}/.docker/cli-plugins" ./cmd/docker-lock
	@echo "installation passed!"
	@echo "install target passed!"

.PHONY: lock
lock:
	@echo "running docker lock generate..."
	@test -e "$${HOME}/.docker/cli-plugins/docker-lock" || (echo "failed - please run 'make install' first"; exit 1)
	@docker lock generate
	@echo "docker lock generate passed!"

.PHONY: unittest
unittest:
	@echo "running unittest target..."
	@echo "running go test's unit tests, writing coverage output to coverage.html..."
	@go test -race ./... -v -count=1 -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "go test passed!"
	@echo "unittest target passed!"

.PHONY: clean
clean:
	@echo "running clean target..."
	@echo "removing docker-lock from docker's cli-plugins folder..."
	@rm -f "$${HOME}"/.docker/cli-plugins/docker-lock*
	@echo "removing passed!"
	@echo "clean target passed!"

.PHONY: inttest
inttest: clean install
	@echo "running inttest target..."
	@./test/demo-app/tests.sh
	@echo "inttest target passed!"
