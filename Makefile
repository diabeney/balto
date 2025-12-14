.PHONY: build test run stop lint lint-fix format install-hooks show-logs

build:
	go build -o bin/balto ./cmd/balto

test:
	go test ./... -v

run:
	docker compose up --build -d

stop:
	docker compose stop

lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix

format:
	go fmt ./...

install-hooks:
	@bash scripts/install-hooks.sh


target ?= "balto"
tail ?= "200"
show-logs:
	@echo "Showing logs for container $(target) with tail $(tail)"
	docker compose logs -f --tail=$(tail) $(target)

