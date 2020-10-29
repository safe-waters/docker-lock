SHELL=/bin/bash -euo pipefail

.PHONY: all
all: clean format lint install unittest

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
	
OSFLAG 				:=
ifeq ($(OS),Windows_NT)
	OSFLAG += windows
else
	UNAME_S := $(shell uname -s)
	ifeq ($(UNAME_S),Linux)
		OSFLAG += linux
	endif
	ifeq ($(UNAME_S),Darwin)
		OSFLAG += mac
	endif
endif

.PHONY: install
install:
	@echo "running build target..."
	@echo "installing docker-lock into docker's cli-plugins folder..."
	@if [[ $(OSFLAG) == "windows" ]]; then \
		mkdir -p ${USERPROFILE}/.docker/cli-plugins; \
		CGO_ENABLED=0 go build -o ${USERPROFILE}/.docker/cli-plugins/docker-lock.exe ./cmd/docker-lock; \
		ls -al ${USERPROFILE};
		ls -al ${USERPROFILE}/.docker;
		ls -al ${USERPROFILE}/.docker/cli-plugins;
		chmod +x ${USERPROFILE}/.docker/cli-plugins/docker-lock.exe; \
	else \
		mkdir -p ${HOME}/.docker/cli-plugins; \
		CGO_ENABLED=0 go build -o ${HOME}/.docker/cli-plugins/docker-lock ./cmd/docker-lock; \
		chmod +x ${HOME}/.docker/cli-plugins/docker-lock; \
	fi
	@echo "installation passed!"
	@echo "build target passed!"

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
	@if [[ $(OSFLAG) == "windows" ]]; then \
		rm -f ${USERPROFILE}/.docker/cli-plugins/docker-lock.exe; \
	else \
		rm -f ${HOME}/.docker/cli-plugins/docker-lock; \
	fi
	@echo "removing passed!"
	@echo "clean target passed!"

.PHONY: inttest
inttest: clean install
	@echo "running inttest target..."
	@./test/registry/firstparty/tests.sh $(OSFLAG) && \
    	./test/registry/contrib/tests.sh && \
    	./test/demo-app/tests.sh;
	@echo "inttest passed!"