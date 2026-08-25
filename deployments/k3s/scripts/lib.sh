#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
K3S_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd "${K3S_ROOT}/../.." && pwd)"
K8S_ROOT="$(cd "${K3S_ROOT}/../k8s" && pwd)"
COMMON="${SCRIPT_DIR}/common.sh"

# shellcheck source=common.sh
source "$COMMON"

load_config() {
  source_deploy_config \
    "${K3S_ROOT}/config.env" \
    "${K3S_ROOT}/../config.env" \
    || true
  MONGO_NFS_SERVER="${MONGO_NFS_SERVER:-192.168.1.100}"
  MONGO_NFS_PATH="${MONGO_NFS_PATH:-/export/reaperc2-metric}"
  MONGO_DATABASE="${MONGO_DATABASE:-reaperc2-metric}"
  MONGO_NODE_HOSTNAME="${MONGO_NODE_HOSTNAME:-control-plane}"
  REGISTRY_SERVER="${REGISTRY_SERVER:-registry.example.com}"
  REGISTRY_SECRET_NAME="${REGISTRY_SECRET_NAME:-registry-credentials}"
  REAPERC2_IMAGE="${REAPERC2_IMAGE:-registry.example.com/reaperc2}"
  REAPERC2_IMAGE_TAG="${REAPERC2_IMAGE_TAG:-latest}"
  REAPERC2_BUILD_ARCH="${REAPERC2_BUILD_ARCH:-arm64}"
  OLLAMA_PULL_MODELS="${OLLAMA_PULL_MODELS:-llama3.2:latest}"
  OLLAMA_NFS_SERVER="${OLLAMA_NFS_SERVER:-192.168.1.100}"
  OLLAMA_NFS_PATH="${OLLAMA_NFS_PATH:-/export/ollama-models}"
  OLLAMA_STORAGE_CLASS="${OLLAMA_STORAGE_CLASS:-}"
  REAPERC2_NAMESPACE="${REAPERC2_NAMESPACE:-reaperc2-ns}"
  INGRESS_HOST="${INGRESS_HOST:-beacons.example.com}"
  CERT_MANAGER_ISSUER="${CERT_MANAGER_ISSUER:-letsencrypt-prod}"
}

reaperc2_image_ref() {
  echo "${REAPERC2_IMAGE}:${REAPERC2_IMAGE_TAG}"
}

apply_k3s_mongo_nfs_pv() {
  patch_mongo_nfs_pv "${K3S_ROOT}/mongodb-nfs-pv.yaml" "$MONGO_NFS_SERVER" "$MONGO_NFS_PATH"
}

apply_k3s_mongo_configmaps() {
  kubectl_cmd create configmap mongo-scripts \
    --from-file=mongo-healthcheck.sh="${K3S_ROOT}/mongo/mongo-healthcheck.sh" \
    -n "$REAPERC2_NAMESPACE" \
    --dry-run=client -o yaml | kubectl_cmd apply -f -
  kubectl_cmd create configmap mongo-init \
    --from-file=init-app-user.sh="${K3S_ROOT}/mongo/init-app-user.sh" \
    --from-file=init-app-user.js="${K3S_ROOT}/mongo/init-app-user.js" \
    -n "$REAPERC2_NAMESPACE" \
    --dry-run=client -o yaml | kubectl_cmd apply -f -
}

apply_k3s_mongodb() {
  local node="${MONGO_NODE_HOSTNAME:-control-plane}"
  sed -e "s|kubernetes.io/hostname: .*|kubernetes.io/hostname: ${node}|" \
    "${K3S_ROOT}/mongodb.yaml" | kubectl_cmd apply -f -
}
