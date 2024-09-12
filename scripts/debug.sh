#!/usr/bin/env bash

set -e

NC='\033[0m'
BLACK='\033[0;30m'
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'

LABEL_SELECTOR='app.kubernetes.io/name=kubernetes-ingress-controller'

OUTPUT_DIR=`mktemp -d`
mkdir -p "${OUTPUT_DIR}/resources"
mkdir -p "${OUTPUT_DIR}/logs"


indent() { sed 's/^/    /'; }

dump_resource() {
	local resource_name="$1"
	local filename="${OUTPUT_DIR}/resources/${resource_name}.yaml"
	echo -e "${BLUE}=============== ${resource_name} => ${filename} ================${NC}" | indent
	kubectl get "$resource_name" 2>&1 | indent
	kubectl get "$resource_name" -o yaml > "$filename" 2>&1
}

echo -e "${BLUE}Saving ngrok custom resources...${NC}"
dump_resource httpsedges
dump_resource tlsedges
dump_resource tcpedges
dump_resource domains
dump_resource ngrokmodulesets
dump_resource ngroktrafficpolicies

echo -e "${BLUE}Saving pod logs...${NC}"
kubectl get pods --all-namespaces --selector "$LABEL_SELECTOR" 2>&1 | indent

kubectl get pods --all-namespaces --selector "$LABEL_SELECTOR" -o=custom-columns='NAMESPACE:.metadata.namespace' --no-headers | uniq | while read namespace; do
	mkdir -p "${OUTPUT_DIR}/logs/${namespace}"
	kubectl get pods --namespace="$namespace" --selector "$LABEL_SELECTOR" -o=custom-columns='NAME:.metadata.name' --no-headers | while read pod; do
		kubectl --namespace="$namespace" logs "$pod" --all-containers=true --tail=-1 > "${OUTPUT_DIR}/logs/${namespace}/${pod}.log" 2>&1
	done
done

echo -e "${GREEN}Saved debug info to ${OUTPUT_DIR}${NC}"
