include build/makelib/common.mk

# CLI version
VERSION_SUFFIX ?= -dev
VERSION ?= 0.0.3${VERSION_SUFFIX} # x-release-please-version

##@ Build Targets
.PHONY: build
build-cli: ## Build CLI
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

manifests:
	@$(INFO) manifests: no-op

generate:
	@$(INFO) generate: no-op

##@ Test Targets

COVER_DIR=_build/cov
COVER_PKGS=$(shell go list ./... | grep -v /tests/) # omit integration tests

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

## Tool versions
DOCKER_VERSION ?= 24.0.6
KIND_VERSION ?= 0.20.0
KUBECTL_VERSION ?= 1.24.10

## Tool binaries
binaries: docker kind kubectl

PLATFORM=$(GOOS)
ifeq ("$(GOOS)", "darwin")
PLATFORM=mac
else ifeq ("$(GOOS)", "windows")
PLATFORM=win
endif

export PATH := $(PATH):$(RUNNER_TOOL_CACHE)

.PHONY: docker
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

.PHONY: kind
kind:
	@if [ "$(GITHUB_ACTIONS)" = "true" ]; then \
		@command -v kind >/dev/null 2>&1 || { \
			echo "Kind not found, downloading..."; \
			curl -Lo $(RUNNER_TOOL_CACHE)/kind https://github.com/kubernetes-sigs/kind/releases/download/v$(KIND_VERSION)/kind-$(GOOS)-$(GOARCH); \
			chmod +x $(RUNNER_TOOL_CACHE)/kind; \
		} \
	fi

.PHONY: kubectl
kubectl:
	@if [ "$(GITHUB_ACTIONS)" = "true" ]; then \
		@command -v kubectl >/dev/null 2>&1 || { \
			echo "Kubectl not found, downloading..."; \
			curl -Lo $(RUNNER_TOOL_CACHE)/kubectl https://dl.k8s.io/release/v$(KUBECTL_VERSION)/bin/$(GOOS)/$(GOARCH)/kubectl; \
			chmod +x $(RUNNER_TOOL_CACHE)/kubectl; \
		} \
	fi
