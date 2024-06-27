# If you update this file, please follow:
# https://www.thapaliya.com/en/writings/well-documented-makefiles/

# Meta
.PHONY: docker kind kubectl helm build test
.DEFAULT_GOAL:=help

# Format output
TIME   = `date +%H:%M:%S`
RED    := $(shell printf "\033[31m")
GREEN  := $(shell printf "\033[32m")
CNone  := $(shell printf "\033[0m")

OK   = echo ${TIME} ${GREEN}[ OK ]${CNone}
ERR  = echo ${TIME} ${RED}[ ERR ]${CNone} "error:"

# Product Version
VERSION_SUFFIX ?= -dev
VERSION ?= 0.0.2${VERSION_SUFFIX} # x-release-please-version

# Common vars
MAKEFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
CURRENT_DIR := $(dir $(MAKEFILE_PATH))
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
	$(foreach platform,$(PLATFORMS),\
		$(eval GOOS=$(word 1,$(subst /, ,$(platform)))) \
		$(eval GOARCH=$(word 2,$(subst /, ,$(platform)))) \
		echo "Building CLI for $(GOOS)/$(GOARCH)..."; \
		CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags " \
			-X github.com/validator-labs/validatorctl/cmd.Version=$(VERSION)" \
			-a -o bin/validator-$(GOOS)-$(GOARCH) validator.go; \
		sha256sum bin/validator-$(GOOS)-$(GOARCH) > bin/validator-$(GOOS)-$(GOARCH).sha256;)

##@ Static Analysis Targets

reviewable: fmt vet lint ## Ensure code is ready for review
	go mod tidy

check-diff: reviewable ## Execute auto-gen code commands and ensure branch is clean
	git --no-pager diff
	git diff --quiet || ($(ERR) please run 'make reviewable' to include all changes && false)
	@$(OK) branch is clean

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

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool versions
DOCKER_VERSION ?= 24.0.6
HELM_VERSION ?= 3.14.0
GOLANGCI_LINT_VERSION ?= v1.59.1
KIND_VERSION ?= 0.20.0
KUBECTL_VERSION ?= 1.24.10

## Tool binaries
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)

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

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint,${GOLANGCI_LINT_VERSION})

gocovmerge:
ifeq (, $(shell which gocovmerge))
	go version
	go install github.com/wadey/gocovmerge@latest
	go mod tidy
GOCOVMERGE=$(GOBIN)/gocovmerge
else
GOCOVMERGE=$(shell which gocovmerge)
endif

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary (ideally with version)
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f $(1) ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv -f "$$(echo "$(1)" | sed "s/-$(3)$$//")" $(1) ;\
}
endef
