
.PHONY: crossplane-composition
crossplane-composition: export KUBECONFIG = $(KIND_KUBECONFIG)
crossplane-composition: local-install .provider-kubernetes ## Installs the Crossplane compositions in kind cluster
	kubectl apply -f crossplane/objectbucket/composite.yaml -f crossplane/objectbucket/composition.yaml
	kubectl create ns provider-cloudscale-secrets --save-config -o yaml --dry-run=client | kubectl apply -f -
	kubectl label namespace default appuio.io/organization=my-org --overwrite

.PHONY: .provider-kubernetes
.provider-kubernetes: export KUBECONFIG = $(KIND_KUBECONFIG)
.provider-kubernetes: crossplane-setup
	kubectl apply -f crossplane/provider-kubernetes.yaml
	@kubectl wait --for condition=Healthy provider.pkg.crossplane.io/provider-kubernetes --timeout 60s
	@kubectl -n crossplane-system wait --for condition=Ready $$(kubectl -n crossplane-system get pods -o name -l pkg.crossplane.io/provider=provider-kubernetes) --timeout 60s

.PHONY: install-samples-composition
install-samples-composition: crossplane-composition provider-config
	yq ./crossplane/samples/*.yaml | kubectl apply -f -
