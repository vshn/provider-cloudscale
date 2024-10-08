## These are some common variables for Make

PROJECT_ROOT_DIR = .

## BUILD:go
BIN_FILENAME ?= provider-cloudscale
go_bin ?= $(PWD)/.work/bin
$(go_bin):
	@mkdir -p $@

## BUILD:docker
DOCKER_CMD ?= docker

IMG_TAG ?= latest
CONTAINER_REGISTRY ?= ghcr.io
UPBOUND_CONTAINER_REGISTRY ?= xpkg.upbound.io

# Image URL to use all building/pushing image targets
LOCAL_PACKAGE_IMG = localhost:5000/$(ORG)/$(APP_NAME)/package:$(IMG_TAG)

## KIND:setup

# https://hub.docker.com/r/kindest/node/tags
KIND_NODE_VERSION ?= v1.28.9
KIND_IMAGE ?= docker.io/kindest/node:$(KIND_NODE_VERSION)
KIND_KUBECONFIG ?= $(kind_dir)/kind-kubeconfig-$(KIND_NODE_VERSION)
KIND_CLUSTER ?= $(APP_NAME)-$(KIND_NODE_VERSION)

# TEST:integration
ENVTEST_ADDITIONAL_FLAGS ?= --bin-dir "$(kind_dir)"
# See https://storage.googleapis.com/kubebuilder-tools/ for list of supported K8s versions
ENVTEST_K8S_VERSION = 1.23.x
INTEGRATION_TEST_DEBUG_OUTPUT ?= false
