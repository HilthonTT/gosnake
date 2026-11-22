# Common development tasks. Run `make help` for the list.

.PHONY: help build run test lint api-test api-run docker clean

help: ## Show this help
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "} {printf "  %-10s %s\n", $$1, $$2}'

build: ## Build the gosnake binary into ./bin
	go build -trimpath -o ./bin/ ./cmd/gosnake

run: ## Run gosnake from source
	go run ./cmd/gosnake

test: ## Run the Go tests with the race detector
	go test -race ./...

lint: ## Run golangci-lint
	golangci-lint run

api-test: ## Build and test the Tips.Api service
	dotnet test tips-api/Tips.slnx

api-run: ## Run the Tips.Api service locally
	dotnet run --project tips-api/Tips.Api

docker: ## Build both Docker images
	docker build -t gosnake .
	docker build -t tips-api tips-api/Tips.Api

clean: ## Remove build output
	rm -rf ./bin ./dist
