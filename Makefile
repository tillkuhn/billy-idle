.PHONY: help
help: ## Shows the help
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
        awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' | sort
	@echo ''

.PHONY: run
run: ## Run tracker
	go run main.go

.PHONY: test
test: ## Test go code
	go test ./...

.PHONY: lint
lint: ## Lint go code
	golangci-lint run

update: ## Update all go dependencies
	@go get -u all