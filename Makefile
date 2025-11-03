.PHONY: build test run

build:
	go build -o bin/balto ./cmd/balto

test:
	go test ./... -v

run:
	go run ./cmd/balto

