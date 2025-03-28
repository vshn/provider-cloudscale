# Set Shell to bash, otherwise some targets fail with dash/zsh etc.
SHELL := /bin/bash

# Disable built-in rules
MAKEFLAGS += --no-builtin-rules
MAKEFLAGS += --no-builtin-variables
.SUFFIXES:
.SECONDARY:
.DEFAULT_GOAL := help

# General variables
include Makefile.vars.mk

# Following includes do not print warnings or error if files aren't found
# Optional Documentation module.
-include docs/antora-preview.mk docs/antora-build.mk
# Optional kind module
-include kind/kind.mk
# Local Env & testing
-include test/local.mk
# Crossplane packaging
-include package/package.mk
# CI automation
-include ci.mk

golangci_bin = $(go_bin)/golangci-lint

.PHONY: help
help: ## Show this help
	@grep -E -h '\s##\s' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: build-bin docker-build ## All-in-one build

.PHONY: build-bin
build-bin: export CGO_ENABLED = 0
build-bin: fmt vet ## Build binary
	env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	go build -o $(BIN_FILENAME) .

.PHONY: test
test: test-go ## All-in-one test

.PHONY: test-go
test-go: ## Run unit tests against code
	go test -race ./...

.PHONY: fmt
fmt: ## Run 'go fmt' against code
	go fmt ./...

.PHONY: vet
vet: ## Run 'go vet' against code
	go vet ./...

.PHONY: lint
lint: generate fmt golangci-lint git-diff ## All-in-one linting

git-diff:
	@echo 'Check for uncommitted changes ...'
	git diff --exit-code

.PHONY: golangci-lint
golangci-lint: $(golangci_bin) ## Run golangci linters
	$(golangci_bin) run --timeout 5m ./...

.PHONY: generate
generate: generate-go generate-docs ## All-in-one code generation

.PHONY: generate-go
generate-go: ## Generate Go artifacts
	@go generate ./...

.PHONY: generate-docs
generate-docs: generate-go ## Generate example code snippets for documentation
	@yq e 'del(.metadata.creationTimestamp) | del(.metadata.generation) | del(.status)' ./samples/cloudscale.crossplane.io_objectsuser.yaml > $(docs_moduleroot_dir)/examples/cloudscale_objectsuser.yaml
	@yq e 'del(.metadata.creationTimestamp) | del(.metadata.generation) | del(.status)' ./samples/cloudscale.crossplane.io_providerconfig.yaml > $(docs_moduleroot_dir)/examples/cloudscale_providerconfig.yaml

.PHONY: install-crd
install-crd: export KUBECONFIG = $(KIND_KUBECONFIG)
install-crd: generate kind-setup ## Install CRDs into cluster
	kubectl apply -f package/crds

.PHONY: install-samples
install-samples: export KUBECONFIG = $(KIND_KUBECONFIG)
install-samples: kind-setup provider-config ## Install samples into cluster
	yq ./samples/*.yaml | kubectl apply -f -

.PHONY: delete-samples
delete-samples: export KUBECONFIG = $(KIND_KUBECONFIG)
delete-samples: kind-setup
	yq ./samples/*.yaml | kubectl delete --ignore-not-found --wait=false -f -

.PHONY: run-operator
run-operator: ## Run in Operator mode against your current kube context
	go run . -v 1 operator

.PHONY: clean
clean: .envtest-clean .e2e-test-clean kind-clean ## Cleans local build artifacts
	rm -rf docs/node_modules $(docs_out_dir) dist .cache .work
	$(DOCKER_CMD) rmi $(IMG) || true

$(golangci_bin): | $(go_bin)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$(go_bin)"
