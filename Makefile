BIN := "./bin/antibf"
BIN_CLI :="./bin/cli"
DOCKER_IMG="antibf:develop"
DSN="postgres://postgres:SecurePass@/OTUSAntibf"

GIT_HASH := $(shell git log --format="%h" -n 1)
LDFLAGS := -X main.release="develop" -X main.buildDate=$(shell date -u +%Y-%m-%dT%H:%M:%S) -X main.gitHash=$(GIT_HASH)

migrate-goose:
	goose --dir=migrations pgx $(DSN) up

build:
	go build -v -o $(BIN) -ldflags "$(LDFLAGS)" ./cmd/antibruteforce
	go build -v -o $(BIN_CLI) -ldflags "$(LDFLAGS)" ./cmd/cli

run-bin: build
	$(BIN) -config ./configs/config.env > antibfLog.txt
	$(BIN_CLI) -config ./configs/config_cli.env > antibfCliLog.txt

build-img:
	docker build --build-arg=LDFLAGS="$(LDFLAGS)" -t $(DOCKER_IMG) -f build/Dockerfile .

run-img:
	docker run $(DOCKER_IMG)

stop-img:
	docker stop $(DOCKER_IMG)

version: build
	$(BIN) version

test:
	CGO_ENABLED=1 go test -race -count 100 ./internal/...

install-lint-dependencies:
	(which golangci-lint > /dev/null) || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/main/install.sh | sh -s -- -b $(go env $GOPATH)/bin v1.64.8

lint: install-lint-dependencies
	golangci-lint run ./...

run: build
	docker-compose -f ./deployments/docker-compose.yaml up --build > deployLog.txt

down:
	docker-compose -f ./deployments/docker-compose.yaml down

integration-tests:
	docker-compose -f ./deployments/docker-compose.yaml -f ./deployments/docker-compose.test.yaml up --build --exit-code-from integration_tests && \
	docker-compose -f ./deployments/docker-compose.yaml -f ./deployments/docker-compose.test.yaml down > deployIntegrationTestsLog.txt

.PHONY: build run-bin build-img run-img stop-img version test lint run down integration-tests