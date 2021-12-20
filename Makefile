SHELL=bash

PKG_LIST := $(shell go list ./...)

all: format lint vet test

format: ## Check coding style
	@DIFF=$$(gofmt -d .); echo -n "$$DIFF"; test -z "$$DIFF"

lint: ## Lint the files
	@golint -set_exit_status ${PKG_LIST}

vet: ## Examine and report suspicious constructs
	@go vet ${PKG_LIST}

test: ## Run unit tests
	@go test ${PKG_LIST}

msgcheck: ## Check the generated messages
	@go generate ./...
	@echo Checking if the generated files were forgotten to commit...
	@DIFF=$$(git diff --cached); echo -n "$$DIFF"; test -z "$$DIFF"

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
