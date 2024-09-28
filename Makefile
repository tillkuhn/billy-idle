ARCH = $(shell uname -m)
BINARY ?= billy
LAUNCHD_LABEL ?= com.github.tillkuhn.billy-idle
PROJECT_PKG = $(shell grep -e ^module go.mod|cut -d' '  -f2|xargs)
# git info for ldflags inspired by https://github.com/oras-project/oras/blob/main/Makefile
GIT_COMMIT = $(shell git rev-parse --short HEAD)
GIT_TAG     = $(shell git describe --tags --abbrev=0 --exact-match 2>/dev/null)
GIT_DIRTY   = $(shell test -n "`git status --porcelain`" && echo "dirty" || echo "clean")
LDFLAGS = -w
LDFLAGS += -X $(PROJECT_PKG)/internal/version.GitTreeState=${GIT_DIRTY}
LDFLAGS += -X $(PROJECT_PKG)/internal/version.GitCommit=${GIT_COMMIT}
ifneq ($(GIT_TAG),)
	LDFLAGS += -X $(PROJECT_PKG)/internal/version.BuildMetadata=$(VERSION)
endif

.PHONY: help
help: ## Shows the help
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
        awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' | sort
	@echo ''

.PHONY: build-mac
build-mac: ## build for mac current arch
	GOARCH=$(ARCH) CGO_ENABLED=0 GOOS=darwin go build -v --ldflags="$(LDFLAGS)" \
		-o bin/darwin/$(ARCH)/$(BINARY) $(CLI_PKG)

run-mac: build-mac ## run mac build
	bin/darwin/$(ARCH)/$(BINARY)

.PHONY: build
build: build-mac ## build all targets
	@find bin -type f

.PHONY: service
install: ## Install as launchd managed service
	mkdir -p $(HOME)/.billy-idle
	cp bin/darwin/$(ARCH)/$(BINARY) $(HOME)/bin/$(BINARY)
	launchctl unload -w ~/Library/LaunchAgents/$(LAUNCHD_LABEL).plist
	cat agent.plist |envsubst '$$HOME' > $(HOME)/Library/LaunchAgents/$(LAUNCHD_LABEL).plist
	launchctl load -w ~/Library/LaunchAgents/$(LAUNCHD_LABEL).plist
	launchctl list $(LAUNCHD_LABEL)
	launchctl start $(LAUNCHD_LABEL)
	sleep 1
	tail $(HOME)/.billy-idle/agent.log

.PHONY: clean
clean: ## Clean output directory
	rm -rf bin/

.PHONY: run
run: ## Run app in tracker mode
	go run main.go -env dev -idle 10s -interval 2s -drop-create true

.PHONY: run-help
run-help: ## Run app in help mode
	@go run main.go help

.PHONY: test
test: ## Runs all tests  (with colorized output support if gotest is installed)
	@if hash gotest 2>/dev/null; then \
	  gotest -v -coverpkg=./... -coverprofile=coverage.out ./...; \
  	else go test -v -coverpkg=./... -coverprofile=coverage.out ./...; fi

#@go tool cover -func coverage.out | grep "total:"
.PHONY: coverage
coverage: test ## Displays coverage per func on cli
	go tool cover -func=coverage.out

.PHONY: lint
lint: ## Lint go code
	@go fmt ./...
	@golangci-lint run --fix

PHONE: update
update: ## Update all go dependencies
	@go get -u all

.PHONY: minor
minor: ## Create Minor Release
	@if hash semtag 2>/dev/null; then \
		semtag final -s minor; \
  	else echo "This target requires semtag, download from https://github.com/nico2sh/semtag"; fi