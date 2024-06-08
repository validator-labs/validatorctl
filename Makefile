# If you update this file, please follow:
# https://www.thapaliya.com/en/writings/well-documented-makefiles/

# Meta
.PHONY: docker kind kubectl helm build test
.DEFAULT_GOAL:=help

# Images
IMAGE_TAG ?= latest

# TODO: update this image location
CLI_IMG ?= "gcr.io/spectro-common-dev/${USER}/validator:$(IMAGE_TAG)"

# Dependency Versions
BUILDER_GOLANG_VERSION ?= 1.22
DOCKER_VERSION ?= 24.0.6
HELM_VERSION ?= 3.14.0
ENVTEST_VERSION ?= 1.27.1
GOLANGCI_VERSION ?= 1.54.2
KIND_VERSION ?= 0.20.0
KUBECTL_VERSION ?= 1.24.10

# Product Version
VERSION_SUFFIX ?= -dev
VERSION ?= 0.0.1${VERSION_SUFFIX}

# Common vars
MAKEFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
CURRENT_DIR := $(dir $(MAKEFILE_PATH))
TEMPLATE_DIR := $(CURRENT_DIR)/server/internal/templates
EMBED_BIN := ./pkg/utils/embed/bin
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

# Swagger vars
MAKE_COMMAND=make -f
INSTALLER_MAKE_PATH := $(CURRENT_DIR)server/internal/spec/Makefile

# Integrated Images List
IMAGE_LIST=_build/images/images.list

##@ Help Targets
help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[0m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build Targets
build: binaries  ## Build CLI
	@echo "Building CLI binary..."
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) GO111MODULE=on go build -ldflags " \
	  -X github.com/validator-labs/validatorctl/cmd.Version=$(VERSION)" \
	  -a -o bin/validator validator.go

build-dev: binaries build-cli  ## Build CLI & copy validator binary into your PATH
	sudo cp bin/validator /usr/local/bin

get-version:  ## Get the product version
	@echo "$(VERSION)"

##@ Static Analysis Targets
fmt:  ## Run go fmt
	go fmt  ./...

lint: golangci-lint ## Run golangci-lint
	$(GOLANGCI_LINT) run

vet: binaries ## Run go vet
	go vet ./...

##@ Test Targets
test-unit: ## Run unit tests
	@mkdir -p $(COVER_DIR)/unit
	rm -rf $(COVER_DIR)/unit/*
	IS_TEST=true CLI_VERSION=$(VERSION) go test -v -race -parallel 6 -timeout 20m \
		-covermode=atomic -coverprofile=$(COVER_DIR)/unit/unit.out $(COVER_PKGS)

# For now we can't enable -race for integration tests
# due to https://github.com/spf13/viper/issues/174
test-integration: binaries init-kubebuilder ## Run integration tests
	@mkdir -p $(COVER_DIR)/integration
	rm -rf $(COVER_DIR)/integration/*
	KUBEBUILDER_ASSETS=${KUBEBUILDER_ASSETS} IS_TEST=true CLI_VERSION=$(VERSION) KUBECONFIG= DISABLE_KIND_CLUSTER_CHECK=true \
		go test -v -parallel 6 -timeout 30m \
		-covermode=atomic -coverpkg=./... -coverprofile=$(COVER_DIR)/integration/integration.out ./tests/...

.PHONY: test
test: gocovmerge test-integration test-unit ## Run unit tests, integration test
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

docker-cli: binaries
	docker buildx build ${BUILD_ARGS} --platform linux/${TARGETARCH} --load -f build/docker/cli.Dockerfile . -t ${CLI_IMG}

docker-compose: binaries  ## Rebuild images and restart docker-compose
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
binaries: docker helm kind kubectl

clean-binaries:  ## Clean embedded binaries
	@echo "Cleaning embedded binaries..."
	rm -rf $(EMBED_BIN)/docker
	rm -rf $(EMBED_BIN)/helm
	rm -rf $(EMBED_BIN)/kind
	rm -rf $(EMBED_BIN)/kubectl

truncate-binaries:
	@echo "Truncating embedded binaries..."
	: > $(EMBED_BIN)/docker
	: > $(EMBED_BIN)/helm
	: > $(EMBED_BIN)/kind
	: > $(EMBED_BIN)/kubectl

docker:
ifeq ("$(wildcard $(EMBED_BIN)/docker)", "")
	if [[ "$(GOOS)" == "windows" ]]; then \
		curl -L https://download.docker.com/$(PLATFORM)/static/stable/x86_64/docker-$(DOCKER_VERSION).zip -o docker.zip; \
		unzip docker.zip; \
		rm -f docker.zip; \
		mv docker/docker.exe $(EMBED_BIN)/docker; \
	else \
		curl -L https://download.docker.com/$(PLATFORM)/static/stable/x86_64/docker-$(DOCKER_VERSION).tgz | tar xz docker/docker; \
		mv docker/docker $(EMBED_BIN)/docker; \
	fi
	chmod +x $(EMBED_BIN)/docker
	rm -rf ./docker
endif

kind:
ifeq ("$(wildcard $(EMBED_BIN)/kind)", "")
	curl -Lo $(EMBED_BIN)/kind https://github.com/kubernetes-sigs/kind/releases/download/v$(KIND_VERSION)/kind-$(GOOS)-$(GOARCH)
	chmod +x $(EMBED_BIN)/kind
endif

kubectl:
ifeq ("$(wildcard $(EMBED_BIN)/kubectl)", "")
	if [[ "$(GOOS)" == "windows" ]]; then \
		curl -Lo $(EMBED_BIN)/kubectl https://dl.k8s.io/release/v$(KUBECTL_VERSION)/bin/$(GOOS)/$(GOARCH)/kubectl.exe; \
	else \
		curl -Lo $(EMBED_BIN)/kubectl https://dl.k8s.io/release/v$(KUBECTL_VERSION)/bin/$(GOOS)/$(GOARCH)/kubectl; \
	fi
	chmod +x $(EMBED_BIN)/kubectl
endif

helm:
ifeq ("$(wildcard $(EMBED_BIN)/helm)", "")
	if [[ "$(GOOS)" == "windows" ]]; then \
		curl -L https://get.helm.sh/helm-v$(HELM_VERSION)-$(GOOS)-$(GOARCH).zip -o helm.zip; \
		unzip helm.zip; \
		rm -f helm.zip; \
		mv windows-amd64/helm.exe $(EMBED_BIN)/helm; \
		rm -rf ./windows-amd64; \
	else \
		curl -L https://get.helm.sh/helm-v$(HELM_VERSION)-$(GOOS)-$(GOARCH).tar.gz | tar xz; \
		mv $(GOOS)-$(GOARCH)/helm $(EMBED_BIN)/helm; \
		rm -rf ./$(GOOS)-$(GOARCH); \
	fi
	chmod +x $(EMBED_BIN)/helm
endif

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

init-kubebuilder: setup-envtest
	$(BIN_DIR)/setup-envtest use --bin-dir $(BIN_DIR) $(ENVTEST_VERSION)
KUBEBUILDER_ASSETS = $(shell pwd)/$(shell $(BIN_DIR)/setup-envtest use -p path --bin-dir $(BIN_DIR) $(ENVTEST_VERSION))

setup-envtest:
ifeq ("$(wildcard $(BIN_DIR)/setup-envtest)", "")
	go get sigs.k8s.io/controller-runtime/tools/setup-envtest
	GOBIN=$(shell pwd)/bin go install sigs.k8s.io/controller-runtime/tools/setup-envtest
endif
SETUP_ENVTEST=$(BIN_DIR)/setup-envtest

