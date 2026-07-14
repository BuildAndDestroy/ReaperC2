#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "${SCRIPT_DIR}/lib.sh"
load_config

DELETE_PVC=false

usage() {
  cat <<'EOF'
Usage: deployments/k3s/scripts/undeploy.sh [options]

Options:
  --delete-pvc   Also delete Mongo and Ollama PVCs (destructive)
  -h, --help     Show help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --delete-pvc) DELETE_PVC=true; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage; exit 1 ;;
  esac
done

kubectl_cmd delete -f "${K3S_ROOT}/service.yaml" --ignore-not-found
kubectl_cmd delete -f "${K3S_ROOT}/deployment.yaml" --ignore-not-found
kubectl_cmd delete -f "${K3S_ROOT}/mongodb.yaml" --ignore-not-found
kubectl_cmd delete -f "${K3S_ROOT}/mongo-secret.yaml" --ignore-not-found
kubectl_cmd delete -f "${K8S_ROOT}/operator-ai.yaml" --ignore-not-found
kubectl_cmd delete -f "${K8S_ROOT}/ollama.yaml" --ignore-not-found
kubectl_cmd delete -f "${K3S_ROOT}/namespace.yaml" --ignore-not-found

if [[ "$DELETE_PVC" == true ]]; then
  kubectl_cmd delete pvc mongo-pvc -n reaperc2-ns --ignore-not-found
  kubectl_cmd delete pvc ollama-data -n ollama-ns --ignore-not-found
  kubectl_cmd delete pv reaperc2-mongo-nfs-pv --ignore-not-found
fi

echo "Undeploy complete."
