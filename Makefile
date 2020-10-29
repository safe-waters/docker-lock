SHELL=/bin/bash -euo pipefail

.PHONY: all
all: clean format lint install test

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
	@echo "running gofmt (without persisting modifications)..."
	@[[ $$(gofmt -s -l . | wc -c) -eq 0 ]];
	@echo "gofmt passed!"
	@echo "running golangci-lint..."
	@golangci-lint run
	@echo "golangci-lint passed!"
	@echo "lint target passed!"
	

.PHONY: install
install:
	@echo "running build target..."
	@echo "installing docker-lock into docker's cli-plugins folder..."
	@mkdir -p ~/.docker/cli-plugins
	@go build -o ~/.docker/cli-plugins ./cmd/docker-lock
	@chmod +x ~/.docker/cli-plugins/docker-lock
	@echo "installation passed!"
	@echo "build target passed!"

.PHONY: test
test:
	@echo "running test target..."
	@echo "running go test's unit tests, writing coverage output to coverage.html..."
	@go test -race ./... -v -count=1 -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "go test passed!"
	@echo "test target passed!"

.PHONY: clean
clean:
	@echo "running clean target..."
	@echo "removing docker-lock from docker's cli-plugins folder..."
	@rm -f ~/.docker/cli-plugins/docker-lock
	@echo "removing passed!"
	@echo "clean target passed!"
