# If you update this file, please follow:
# https://www.thapaliya.com/en/writings/well-documented-makefiles/

# Meta
.PHONY: docker kind kubectl helm build test
.DEFAULT_GOAL:=help

# Dependency Versions
DOCKER_VERSION ?= 24.0.6
HELM_VERSION ?= 3.14.0
GOLANGCI_VERSION ?= 1.54.2
KIND_VERSION ?= 0.20.0
KUBECTL_VERSION ?= 1.24.10

# Product Version
VERSION_SUFFIX ?= -dev
VERSION ?= 0.0.2${VERSION_SUFFIX} # x-release-please-version

# Common vars
MAKEFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
CURRENT_DIR := $(dir $(MAKEFILE_PATH))
BIN_DIR ?= ./bin
export PATH := $(PATH):$(RUNNER_TOOL_CACHE)

# Go env vars
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

PLATFORM=$(GOOS)
ifeq ("$(GOOS)", "darwin")
PLATFORM=mac
else ifeq ("$(GOOS)", "windows")
PLATFORM=win
endif
TARGETARCH ?= amd64

# Test vars
COVER_DIR=_build/cov
COVER_PKGS=$(shell go list ./... | grep -v /tests/) # omit integration tests

##@ Help Targets
help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[0m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build Targets
build: ## Build CLI
	@echo "Building CLI binary..."
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags " \
	  -X github.com/validator-labs/validatorctl/cmd.Version=$(VERSION)" \
	  -a -o bin/validator validator.go

PLATFORMS ?= linux/amd64 darwin/arm64 windows/amd64
build-release:  ## Build CLI for multiple platforms
	for platform in $(PLATFORMS); do \
		platform_split=($${platform//\// }); \
		GOOS=$${platform_split[0]}; \
		GOARCH=$${platform_split[1]}; \
		echo "Building CLI for $${GOOS}/$${GOARCH}..."; \
		CGO_ENABLED=0 GOOS=$${GOOS} GOARCH=$${GOARCH} go build -ldflags " \
		  -X github.com/validator-labs/validatorctl/cmd.Version=$(VERSION)" \
		  -a -o bin/validator-$${GOOS}-$${GOARCH} validator.go; \
	done

##@ Static Analysis Targets
fmt:  ## Run go fmt
	go fmt  ./...

lint: golangci-lint ## Run golangci-lint
	$(GOLANGCI_LINT) run

vet: ## Run go vet
	go vet ./...

##@ Test Targets
test-unit: ## Run unit tests
	@mkdir -p $(COVER_DIR)/unit
	rm -rf $(COVER_DIR)/unit/*
	IS_TEST=true CLI_VERSION=$(VERSION) go test -v -parallel 6 -timeout 20m \
		-covermode=atomic -coverprofile=$(COVER_DIR)/unit/unit.out $(COVER_PKGS)

# For now we can't enable -race for integration tests
# due to https://github.com/spf13/viper/issues/174
test-integration: ## Run integration tests
	@mkdir -p $(COVER_DIR)/integration
	rm -rf $(COVER_DIR)/integration/*
	IS_TEST=true CLI_VERSION=$(VERSION) KUBECONFIG= DISABLE_KIND_CLUSTER_CHECK=true \
		go test -v -parallel 6 -timeout 30m \
		-covermode=atomic -coverpkg=./... -coverprofile=$(COVER_DIR)/integration/integration.out ./tests/...

.PHONY: test
test: binaries gocovmerge test-unit test-integration ## Run unit tests, integration test
	$(GOCOVMERGE) $(COVER_DIR)/unit/*.out $(COVER_DIR)/integration/*.out > $(COVER_DIR)/coverage.out.tmp
	# Omit test code from coverage report
	cat $(COVER_DIR)/coverage.out.tmp | grep -vE 'tests' > $(COVER_DIR)/coverage.out
	go tool cover -func=$(COVER_DIR)/coverage.out -o $(COVER_DIR)/cover.func
	go tool cover -html=$(COVER_DIR)/coverage.out -o $(COVER_DIR)/cover.html
	go tool cover -func $(COVER_DIR)/coverage.out | grep total
	cp $(COVER_DIR)/coverage.out cover.out

coverage: ## Show global test coverage
	go tool cover -func $(COVER_DIR)/coverage.out

coverage-html: ## Open global test coverage report in your browser
	go tool cover -html $(COVER_DIR)/coverage.out

coverage-unit: ## Show unit test coverage
	go tool cover -func $(COVER_DIR)/unit/unit.out

coverage-unit-html: ## Open unit test coverage report in your browser
	go tool cover -html $(COVER_DIR)/unit/unit.out

coverage-integration: ## Show integration test coverage
	go tool cover -func $(COVER_DIR)/integration/integration.out

coverage-integration-html: ## Open integration test coverage report in your browser
	go tool cover -html $(COVER_DIR)/integration/integration.out

##@ Tools Targets
binaries: docker helm kind kubectl

docker:
	@if [ "$(GITHUB_ACTIONS)" = "true" ]; then \
		@command -v docker >/dev/null 2>&1 || { \
			echo "Docker not found, downloading..."; \
			curl -L https://download.docker.com/$(PLATFORM)/static/stable/x86_64/docker-$(DOCKER_VERSION).tgz | tar xz docker/docker; \
			mv docker/docker $(RUNNER_TOOL_CACHE)/docker; \
			chmod +x $(RUNNER_TOOL_CACHE)/docker; \
			rm -rf ./docker; \
		} \
	fi

kind:
	@if [ "$(GITHUB_ACTIONS)" = "true" ]; then \
		@command -v kind >/dev/null 2>&1 || { \
			echo "Kind not found, downloading..."; \
			curl -Lo $(RUNNER_TOOL_CACHE)/kind https://github.com/kubernetes-sigs/kind/releases/download/v$(KIND_VERSION)/kind-$(GOOS)-$(GOARCH); \
			chmod +x $(RUNNER_TOOL_CACHE)/kind; \
		} \
	fi

kubectl:
	@if [ "$(GITHUB_ACTIONS)" = "true" ]; then \
		@command -v kubectl >/dev/null 2>&1 || { \
			echo "Kubectl not found, downloading..."; \
			curl -Lo $(RUNNER_TOOL_CACHE)/kubectl https://dl.k8s.io/release/v$(KUBECTL_VERSION)/bin/$(GOOS)/$(GOARCH)/kubectl; \
			chmod +x $(RUNNER_TOOL_CACHE)/kubectl; \
		} \
	fi

helm:
	@if [ "$(GITHUB_ACTIONS)" = "true" ]; then \
		@command -v helm >/dev/null 2>&1 || { \
			echo "Helm not found, downloading..."; \
			curl -L https://get.helm.sh/helm-v$(HELM_VERSION)-$(GOOS)-$(GOARCH).tar.gz | tar xz; \
			mv $(GOOS)-$(GOARCH)/helm $(RUNNER_TOOL_CACHE)/helm; \
			rm -rf ./$(GOOS)-$(GOARCH); \
			chmod +x $(RUNNER_TOOL_CACHE)/helm; \
		} \
	fi

golangci-lint:
	if ! test -f $(BIN_DIR)/golangci-lint-linux-amd64; then \
		curl -LOs https://github.com/golangci/golangci-lint/releases/download/v$(GOLANGCI_VERSION)/golangci-lint-$(GOLANGCI_VERSION)-linux-amd64.tar.gz; \
		tar -zxf golangci-lint-$(GOLANGCI_VERSION)-linux-amd64.tar.gz; \
		mv golangci-lint-$(GOLANGCI_VERSION)-*/golangci-lint $(BIN_DIR)/golangci-lint-linux-amd64; \
		chmod +x $(BIN_DIR)/golangci-lint-linux-amd64; \
		rm -rf ./golangci-lint-$(GOLANGCI_VERSION)-linux-amd64*; \
	fi
	if ! test -f $(BIN_DIR)/golangci-lint-$(GOOS)-$(GOARCH); then \
		curl -LOs https://github.com/golangci/golangci-lint/releases/download/v$(GOLANGCI_VERSION)/golangci-lint-$(GOLANGCI_VERSION)-$(GOOS)-$(GOARCH).tar.gz; \
		tar -zxf golangci-lint-$(GOLANGCI_VERSION)-$(GOOS)-$(GOARCH).tar.gz; \
		mv golangci-lint-$(GOLANGCI_VERSION)-*/golangci-lint $(BIN_DIR)/golangci-lint-$(GOOS)-$(GOARCH); \
		chmod +x $(BIN_DIR)/golangci-lint-$(GOOS)-$(GOARCH); \
		rm -rf ./golangci-lint-$(GOLANGCI_VERSION)-$(GOOS)-$(GOARCH)*; \
	fi
GOLANGCI_LINT=$(BIN_DIR)/golangci-lint-$(GOOS)-$(GOARCH)

gocovmerge:
ifeq (, $(shell which gocovmerge))
	go version
	go install github.com/wadey/gocovmerge@latest
	go mod tidy
GOCOVMERGE=$(GOBIN)/gocovmerge
else
GOCOVMERGE=$(shell which gocovmerge)
endif
