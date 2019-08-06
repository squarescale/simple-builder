.DEFAULT_GOAL := build
VERSION = $(shell git describe --always --dirty)
GOTAGS ?= sqs

deps: ## Dependencies
	go get ./...

build: deps ## Build
	go build -tags "$(GOTAGS)" -ldflags "-X main.version=$(VERSION)" .

test: ## Run tests
	go test ./...

help: ## This help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

get-dep: always
	go get -u github.com/golang/dep/cmd/dep

dep-status: always ## Show the status of vendored dependencies
	@if !which dep >/dev/null 2>&1; then $(MAKE) get-dep; fi
	dep status

dep-update: always ## Update vendored dependencies to the latest possible version
	@if !which dep >/dev/null 2>&1; then $(MAKE) get-dep; fi
	dep ensure -update

vendor: always ## Vendor missing dependencies
	@if !which dep >/dev/null 2>&1; then $(MAKE) get-dep; fi
	dep ensure -v -vendor-only

dep-check: always ## Check vendored dependencies
	@if !which dep >/dev/null 2>&1; then $(MAKE) get-dep; fi
	dep ensure -dry-run

always:

.PHONY: always test help build
