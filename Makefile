# If you update this file, please follow:
# https://www.thapaliya.com/en/writings/well-documented-makefiles/

# Meta
.PHONY: docker kind kubectl helm build test
.DEFAULT_GOAL:=help

# Images
IMAGE_TAG ?= latest
CLI_IMG ?= "quay.io/validator-labs/validatorctl:$(IMAGE_TAG)"

# Dependency Versions
BUILDER_GOLANG_VERSION ?= 1.22
DOCKER_VERSION ?= 24.0.6
HELM_VERSION ?= 3.14.0
GOLANGCI_VERSION ?= 1.54.2
KIND_VERSION ?= 0.20.0
KUBECTL_VERSION ?= 1.24.10

# Product Version
VERSION_SUFFIX ?= -dev
VERSION ?= 0.0.1${VERSION_SUFFIX}

# Common vars
MAKEFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
CURRENT_DIR := $(dir $(MAKEFILE_PATH))
BIN_DIR ?= ./bin

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

# Integrated Images List
IMAGE_LIST=_build/images/images.list

##@ Help Targets
help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[0m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build Targets
build: ## Build CLI
	@echo "Building CLI binary..."
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) GO111MODULE=on go build -ldflags " \
	  -X github.com/validator-labs/validatorctl/cmd.Version=$(VERSION)" \
	  -a -o bin/validator validator.go

get-version:  ## Get the product version
	@echo "$(VERSION)"

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
test: gocovmerge test-unit test-integration ## Run unit tests, integration test
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

##@ Image Targets

BUILD_ARGS = --build-arg CLI_VERSION=${VERSION} --build-arg BUILDER_GOLANG_VERSION=${BUILDER_GOLANG_VERSION}

docker-all: docker-cli docker-push  ## Builds & pushes Docker images to container registry

docker-cli:
	docker buildx build ${BUILD_ARGS} --platform linux/${TARGETARCH} --load -f build/docker/cli.Dockerfile . -t ${CLI_IMG}

docker-compose: ## Rebuild images and restart docker-compose
	docker compose build
	docker compose up

docker-push: ## Pushes Docker images to container registry
	docker push ${CLI_IMG}
	echo cli,core,${CLI_IMG} >> ${IMAGE_LIST}

docker-rmi:  ## Remove Docker images from local Docker engine
	docker rmi -f ${CLI_IMG}

create-images-list: ## Create the image list for CICD
	mkdir -p _build/images
	touch $(IMAGE_LIST)


##@ Tools Targets
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
