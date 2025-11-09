.PHONY: build test run lint lint-fix format install-hooks

build:
	go build -o bin/balto ./cmd/balto

test:
	go test ./... -v

run:
	go run ./cmd/balto

lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix

format:
	go fmt ./...

install-hooks:
	@bash scripts/install-hooks.sh

