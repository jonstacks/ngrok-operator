##@ Generated Code/Files

.PHONY: generate
generate: controller-gen generate-mocks ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="$(CONTROLLER_GEN_PATHS)"

.PHONY: generate-mocks
generate-mocks:
	go generate ./...

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=ngrok-operator-manager-role crd webhook paths="$(CONTROLLER_GEN_PATHS)" \
		output:crd:artifacts:config=$(HELM_TEMPLATES_DIR)/crds \
		output:rbac:artifacts:config=$(HELM_TEMPLATES_DIR)/rbac


.PHONY: helm-update-snapshots
helm-update-snapshots: _helm_setup ## Update helm unittest snapshots
	$(MAKE) -C $(HELM_CHART_DIR) update-snapshots


helm-update-snapshots-no-deps: ## Update helm unittest snapshots without rebuilding dependencies
	$(MAKE) -C $(HELM_CHART_DIR) update-snapshots
