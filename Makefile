.PHONY: build test run lint lint-fix format install-hooks show-balto-logs

build:
	go build -o bin/balto ./cmd/balto

test:
	go test ./... -v

run:
	docker compose up --build -d

lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix

format:
	go fmt ./...

install-hooks:
	@bash scripts/install-hooks.sh

show-balto-logs:
	docker compose logs -f --tail=200 balto

