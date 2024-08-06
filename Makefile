include build/makelib/common.mk

# CLI version
VERSION_SUFFIX ?= -dev
VERSION ?= 0.1.0${VERSION_SUFFIX} # x-release-please-version

##@ Build Targets

.PHONY: build
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

manifests:
	@$(INFO) manifests: no-op

reviewable-ext:
	@$(INFO) Checking for validator version updates...
	bash hack/update-versions.sh

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

binaries: kind kubectl helm

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
