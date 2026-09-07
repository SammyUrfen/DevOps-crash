BIN := bin/server

.PHONY: help db run build test lint check clean

help:  ## Show the targets
	@grep -E '^[a-z-]+:.*?## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "%-8s %s\n", $$1, $$2}'

db:  ## Start Postgres in Docker
	docker compose up -d

run: db  ## Start Postgres, then run the server
	go run ./cmd/server

build:  ## Compile the server
	go build -o $(BIN) ./cmd/server

test:  ## Run the unit tests under the race detector
	go test -race -count=1 ./...

lint:  ## Format and vet
	gofmt -l .
	go vet ./...

check: lint test  ## Run lint and test

clean:  ## Remove the binary and stop Postgres
	rm -rf bin
	docker compose down
