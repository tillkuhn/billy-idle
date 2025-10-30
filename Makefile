#-------------------
# Common Variables
#-------------------
ARCH = $(shell uname -m)
OSNAME = $(shell uname -o)
PROJECT_PKG = $(shell grep -e ^module go.mod|cut -d' '  -f2|xargs)
# git info for ldflags inspired by https://github.com/oras-project/oras/blob/main/Makefile
GIT_COMMIT = $(shell git rev-parse --short HEAD)
GIT_TAG    = $(shell git describe --tags --abbrev=0 2>/dev/null)
#GIT_DIRTY   = $(shell test -n "`git status --porcelain`" && echo "dirty" || echo "clean")
LDFLAGS = -w
LDFLAGS += -X main.cmmit=${GIT_COMMIT}
LDFLAGS += -X main.date=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

ifneq ($(GIT_TAG),)
	LDFLAGS += -X main.version=$(GIT_TAG)
endif
ifeq ($(OSNAME),Darwin)
  OS = darwin
  # on MacOS, GOARCH should be amd64, not x86_64 returned by uname -m
  ifeq ($(ARCH),x86_64)
	ARCH = amd64
  endif
else
  OS = linux
endif

#-------------------
# Custom Variables
#-------------------

APP_NAME=billy-idle
BINARY ?= billy
DEFAULT_ENV ?= default
DEV_PORT ?= 50052
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
build-mac: grpc-gen  ## build for mac current arch using default goreleaser target path
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

# $(shell go list ./... | grep -v internal/pb)
.PHONY: test
test: lint ## Run tests with coverage, implies lint, excludes generated *.pb.go files
	@if hash gotest 2>/dev/null; then \
	  gotest -v -coverpkg=./... -coverprofile=coverage.out.tmp  ./... ; \
	else go test -v -coverpkg=./... -coverprofile=coverage.out.tmp ./... ; fi
	grep -v ".pb.go" coverage.out.tmp > coverage.out
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
run: ## Run app in tracker mode (dev env), add -drop-create to recreate db
	go run main.go --debug track --env dev --idle 10s --interval 5s --port $(DEV_PORT)

.PHONY: punch
punch: ## Show punch clock report for default db
	go run main.go --debug punch --env $(DEFAULT_ENV)

.PHONY: rm
rm: ## Run rm command
	go run main.go --debug rm 1234

.PHONY: wsp
wsp: ## Show status using gRPC Client
	go run main.go --debug wsp --port $(DEV_PORT)

.PHONY: report
report: ## Show report for dev env db
	go run main.go --debug report --env dev

.PHONY: report-default
report-default: ## Show report for default db
	go run main.go --debug report --env $(DEFAULT_ENV)

.PHONY: run-help
run-help: ## Run app in help mode
	go run main.go
	@echo "----------------------"
	go run main.go track --help
	@echo "----------------------"
	go run main.go report --help
	@echo "----------------------"
	go run main.go punch --help
	@echo "----------------------"

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


# https://grpc.io/docs/languages/go/basics/ + https://grpc.io/docs/languages/go/quickstart/
# for compiler plugins, make sure "$PATH:$(go env GOPATH)/bin" is on your path
.PHONY: grpc-install
grpc-install: ## Installs protobuf and Go gRPC compiler plugins
	brew list --versions protobuf || brew install protobuf
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@protoc-gen-go --version || echo 'Make sure GOPATH/bin is on your PATH'
	@protoc-gen-go-grpc --version
	go get -u google.golang.org/grpc


.PHONY: grpc-gen
grpc-gen: ## Generate gRPC Code with protoc
	protoc --go_out=. --go_opt=paths=source_relative \
	  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	  internal/pb/billy.proto

.PHONY: logs
logs: ## Show agent logs
	@tail -120 $(HOME)/.billy-idle/default/agent.log


.PHONY: minor
minor: ## Create Minor Release
	@if hash semtag 2>/dev/null; then \
		semtag final -s minor; \
  	else echo "This target requires semtag, download from https://github.com/nico2sh/semtag"; fi

.PHONY: usage
usage: ## call main program with --help to display usage (adoc format) and create usage.adoc
	@echo "----" > docs/usage-track.adoc
	@go run main.go track --help | tee -a docs/usage-track.adoc
	@echo "----" >> docs/usage-track.adoc

	@echo "----" > docs/usage-punch.adoc
	@go run main.go punch --help | tee -a docs/usage-punch.adoc
	@echo "----" >> docs/usage-punch.adoc

	@echo "----" > docs/usage-report.adoc
	@go run main.go report --help | tee -a docs/usage-report.adoc
	@echo "----" >> docs/usage-report.adoc
