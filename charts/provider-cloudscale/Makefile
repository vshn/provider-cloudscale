rbac_gen_src = ../../package/rbac/role.yaml
rbac_gen_tgt = templates/operator-clusterrole.yaml

ifeq ($(shell uname -s),Darwin)
	sed := gsed -i
else
	sed := sed -i
endif

$(rbac_gen_tgt):
	@cp $(rbac_gen_src) $@
	@yq -i e '.metadata.name="{{ include \"provider-cloudscale.fullname\" . }}-manager", del(.metadata.creationTimestamp)' $@
	@yq -i e '.metadata.labels.replace="LABELS"' $@
	@$(sed) -e 's/replace: LABELS/{{- include "provider-cloudscale.labels" . | nindent 4 }}/g' $@

.PHONY: prepare
prepare: $(rbac_gen_tgt) ## Helmify generated artifacts

.PHONY: clean
clean: ## Clean generated artifacts
	rm -rf $(rbac_gen_tgt)
