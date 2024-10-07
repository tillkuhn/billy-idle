#-------------------
# Common Variables
#-------------------
ARCH = $(shell uname -m)
OSNAME = $(shell uname -o)
PROJECT_PKG = $(shell grep -e ^module go.mod|cut -d' '  -f2|xargs)
# git info for ldflags inspired by https://github.com/oras-project/oras/blob/main/Makefile
GIT_COMMIT = $(shell git rev-parse --short HEAD)
GIT_TAG     = $(shell git describe --tags --abbrev=0 2>/dev/null)
#GIT_DIRTY   = $(shell test -n "`git status --porcelain`" && echo "dirty" || echo "clean")
LDFLAGS = -w
LDFLAGS += -X main.cmmit=${GIT_COMMIT}
LDFLAGS += -X main.date=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

ifneq ($(GIT_TAG),)
	LDFLAGS += -X main.version=$(GIT_TAG)
endif
ifeq ($(OSNAME),Darwin)
  OS = darwin
else
  OS = linux
endif

#-------------------
# Custom Variables
#-------------------

APP_NAME=billy-idle
BINARY ?= billy
LAUNCHD_LABEL ?= com.github.tillkuhn.$(APP_NAME)

#-------------------
# Common Targets
#-------------------
.PHONY: help
help: ## Shows the help
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
        awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' | sort
	@echo ''

# default goreleaser  dist/billy-idle_darwin_arm64/billy
.PHONY: build-mac
build-mac: ## build for mac current arch using default goreleaser target path
	GOARCH=$(ARCH) CGO_ENABLED=0 GOOS=darwin \
	go build -v --ldflags="$(LDFLAGS)" \
	-o dist/$(APP_NAME)_$(OS)_$(ARCH)/$(BINARY)

run-mac: build-mac ## run mac build
	dist/$(APP_NAME)_$(OS)_$(ARCH)/$(BINARY)

.PHONY: build
build: build-mac ## build all targets
	@find dist -type f -exec ls -hl {} \;

.PHONY: release
release: ## run goreleaser in snapshot mode
	goreleaser build --clean --snapshot

.PHONY: clean
clean: ## Clean output directory
	rm -rf dist/

lint: ## Lint go code
	@go fmt ./...
	@golangci-lint run --fix

.PHONY: test
test: lint ## Run tests with coverage, implies lint
	@if hash gotest 2>/dev/null; then \
	  gotest -v -coverpkg=./... -coverprofile=coverage.out ./...; \
	else go test -v -coverpkg=./... -coverprofile=coverage.out ./...; fi
	@go tool cover -func coverage.out | grep "total:"
	go tool cover -html=coverage.out -o coverage.html
	@echo For coverage report open coverage.html

.PHONY: tidy
tidy: ## Add missing and remove unused modules
	go mod tidy

.PHONY: update
update: ## Update all go dependencies
	@go get -u all

.PHONY: vulncheck
vulncheck: ## Run govulncheck scanner
	@if ! hash govulncheck 2>/dev/null; then \
  		echo "Installing govulncheck"; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	govulncheck ./...

#-------------------
# Custom Targets
#-------------------

.PHONY: run
run: ## Run app in tracker mode, add -drop-create to recreate db
	go run main.go track -env dev -idle 10s -interval 5s -debug

.PHONY: report-dev
report-dev: ## Show report for dev db
	go run main.go report -env dev -debug

.PHONY: report
report: ## Show report for default db
	go run main.go report -debug

.PHONY: run-help
run-help: ## Run app in help mode
	go run main.go help
	go run main.go track -h

.PHONY: install
install: clean build ## Install as launchd managed service
	@mkdir -p $(HOME)/.billy-idle
	@if launchctl list $(LAUNCHD_LABEL) 2>/dev/null|grep '"Program"'; then \
  		echo "$(LAUNCHD_LABEL) is loaded, trigger unload"; \
		launchctl unload -w ~/Library/LaunchAgents/$(LAUNCHD_LABEL).plist; \
	fi
	cp dist/$(APP_NAME)_$(OS)_$(ARCH)/$(BINARY) $(HOME)/bin/$(BINARY)
	cat agent.plist |envsubst '$$HOME' > $(HOME)/Library/LaunchAgents/$(LAUNCHD_LABEL).plist
	launchctl load -w ~/Library/LaunchAgents/$(LAUNCHD_LABEL).plist
	launchctl list $(LAUNCHD_LABEL) | grep '"PID"'
	@sleep 1
	@ps -ef |grep -v grep |grep $(HOME)/bin/billy
	@tail $(HOME)/.billy-idle/default/agent.log


.PHONY: logs
logs: ## Show agent logs
	@tail -120 $(HOME)/.billy-idle/default/agent.log


.PHONY: minor
minor: ## Create Minor Release
	@if hash semtag 2>/dev/null; then \
		semtag final -s minor; \
  	else echo "This target requires semtag, download from https://github.com/nico2sh/semtag"; fi