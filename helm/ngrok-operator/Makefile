.PHONY: _setup
_setup:
	../../scripts/helm-setup.sh
	helm repo add bitnami https://charts.bitnami.com/bitnami
	helm dependency build .

.PHONY: lint
lint: _setup ## Lint the helm chart
	helm lint .

.PHONY: template
template: ## Render the helm chart
	helm template ngrok-operator . --debug

.PHONY: test
test: _setup ## Run helm unittest plugin
	helm unittest -f 'tests/**/*_test.yaml' .

.PHONY: update-snapshots
update-snapshots: _setup ## Update helm unittest snapshots
	helm unittest -f 'tests/**/*_test.yaml' --update-snapshot .
