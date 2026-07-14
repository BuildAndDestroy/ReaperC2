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
  MONGO_NFS_PATH="${MONGO_NFS_PATH:-/volume1/reaperc2db}"
  REGISTRY_SERVER="${REGISTRY_SERVER:-registry.example.com}"
  REGISTRY_SECRET_NAME="${REGISTRY_SECRET_NAME:-registry-credentials}"
  REAPERC2_IMAGE="${REAPERC2_IMAGE:-registry.example.com/reaperc2}"
  REAPERC2_IMAGE_TAG="${REAPERC2_IMAGE_TAG:-latest}"
  OLLAMA_PULL_MODELS="${OLLAMA_PULL_MODELS:-llama3.2:latest}"
  REAPERC2_NAMESPACE="${REAPERC2_NAMESPACE:-reaperc2-ns}"
}

reaperc2_image_ref() {
  echo "${REAPERC2_IMAGE}:${REAPERC2_IMAGE_TAG}"
}

apply_k3s_mongo_nfs_pv() {
  patch_mongo_nfs_pv "${K3S_ROOT}/mongodb-nfs-pv.yaml" "$MONGO_NFS_SERVER" "$MONGO_NFS_PATH"
}
