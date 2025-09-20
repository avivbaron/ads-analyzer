# .PHONY: run test race vet tidy build docker docker-run compose-up compose-down fmt

# run:
# 	LOG_LEVEL=debug PORT=8080 go run ./cmd/server

# test:
# 	go test ./...

# race:
# 	go test ./... -race

# vet:
# 	go vet ./...

# tidy:
# 	go mod tidy

# build:
# 	go build ./cmd/server

# docker:
# 	docker build -t ads-analyzer:latest .

# docker-run:
# 	docker run --rm -p 8080:8080 -e LOG_LEVEL=info ads-analyzer:latest

# compose-up:
# 	docker compose up --build

# compose-down:
# 	docker compose down

# fmt:
# 	go fmt ./...

# -------- ads-analyzer Makefile --------
SHELL := bash

# Discover module path dynamically (fallback if not a module yet)
GO_MODULE := $(shell $(GO) list -m 2>/dev/null || echo github.com/avivbaron/ads-analyzer)

# Build metadata
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
GIT_TAG    := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# ldflags (fills internal/buildinfo variables)
LDFLAGS := -X '$(GO_MODULE)/internal/buildinfo.Version=$(GIT_TAG)' \
           -X '$(GO_MODULE)/internal/buildinfo.Commit=$(GIT_COMMIT)' \
           -X '$(GO_MODULE)/internal/buildinfo.BuildTime=$(BUILD_TIME)'

BIN := ./bin/ads-analyzer


help: ## List targets
	@grep -E '^[a-zA-Z0-9_\-]+:.*?## ' $(MAKEFILE_LIST) | sed -E 's/:.*?## / | /' | column -s '|' -t

tidy: ## go mod tidy
	go mod tidy

fmt: ## go fmt
	go fmt ./...

vet: ## go vet
	go vet ./...

test: ## run unit tests
	go test ./...

race: ## run tests with race detector
	go test ./... -race -count=1

build: ## build binary into ./bin
	mkdir -p bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/server

run: ## run locally with ldflags
	LOG_LEVEL=debug PORT=8080 go run -ldflags "$(LDFLAGS)" ./cmd/server

docker: ## build docker image (passes LDFLAGS)
	docker build --build-arg LDFLAGS="$(LDFLAGS)" -t ads-analyzer:latest .

docker-run: docker ## run docker image
	docker run --name ads-analyzer --rm -p 8080:8080 -e LOG_LEVEL=info ads-analyzer:latest

compose-up: ## docker compose up
	docker compose up --build

compose-down: ## docker compose down
	docker compose down

print-ldflags: ## echo computed ldflags
	@echo $(LDFLAGS)

clean: ## remove build artifacts
	rm -rf bin

.PHONY: clean print-ldflags help tidy fmt vet test race build run docker docker-run compose-up compose-down