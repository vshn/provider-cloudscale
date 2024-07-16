
mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
package_dir := $(notdir $(patsubst %/,%,$(dir $(mkfile_path))))

crossplane_bin = $(go_bin)/kubectl-crossplane
up_bin = $(go_bin)/up

# Build kubectl-crossplane plugin
$(crossplane_bin):export GOBIN = $(go_bin)
$(crossplane_bin): | $(go_bin)
	go install github.com/crossplane/crossplane/cmd/crank@latest
	@mv $(go_bin)/crank $@

# Install up plugin
$(up_bin):export GOBIN = $(go_bin)
$(up_bin): | $(go_bin)
	curl -sL "https://cli.upbound.io" | sh
	@mv up $@

.PHONY: package
package: ## All-in-one packaging and releasing
package: package-push

.PHONY: package-provider-local
package-provider-local: export CONTROLLER_IMG = $(CONTAINER_IMG)
package-provider-local: $(crossplane_bin) generate-go ## Build Crossplane package for local installation in kind-cluster
	@rm -rf package/*.xpkg
	@yq e '.spec.controller.image=strenv(CONTROLLER_IMG)' $(package_dir)/crossplane.yaml.template > $(package_dir)/crossplane.yaml
	@go run github.com/crossplane/crossplane/cmd/crank@latest xpkg build -f$(package_dir)
	@echo Package file: $$(ls $(package_dir)/*.xpkg)

.PHONY: package-provider
package-provider: export CONTROLLER_IMG = $(CONTAINER_IMG)
package-provider: $(up_bin) generate-go build-docker ## Build Crossplane package for Upbound Marketplace
	@rm -rf package/*.xpkg
	@yq e 'del(.spec)' $(package_dir)/crossplane.yaml.template > $(package_dir)/crossplane.yaml
	$(up_bin) xpkg build -f $(package_dir) -o $(package_dir)/provider-cloudscale.xpkg --controller=$(CONTROLLER_IMG)

.PHONY: .local-package-push
.local-package-push: pkg_file = $(shell ls $(package_dir)/*.xpkg)
.local-package-push: $(crossplane_bin) package-provider-local
	@go run github.com/crossplane/crossplane/cmd/crank@latest xpkg push -f $(pkg_file) $(LOCAL_PACKAGE_IMG)

.PHONY: .ghcr-package-push
.ghcr-package-push: pkg_file = $(package_dir)/provider-cloudscale.xpkg
.ghcr-package-push: $(crossplane_bin) package-provider
	@go run github.com/crossplane/crossplane/cmd/crank@latest xpkg push -f $(pkg_file) $(GHCR_PACKAGE_IMG)

.PHONY: .upbound-package-push
.upbound-package-push: pkg_file = $(package_dir)/provider-cloudscale.xpkg
.upbound-package-push: package-provider
	@go run github.com/crossplane/crossplane/cmd/crank@latest xpkg push -f $(pkg_file) $(UPBOUND_PACKAGE_IMG)

.PHONY: package-push
package-push: pkg_file = $(package_dir)/provider-cloudscale.xpkg
package-push: .ghcr-package-push .upbound-package-push ## Push Crossplane package to container registry

.PHONY: .package-clean
.package-clean:
	rm -f $(crossplane_bin) $(up_bin) package/*.xpkg $(package_dir)/crossplane.yaml
