.DEFAULT_GOAL := build
VERSION = $(shell git describe --always --dirty)

build: ## Build
	go build -ldflags "-X main.version=$(VERSION)" .

test: ## Run tests
	go test ./...

help: ## This help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: always test help build
