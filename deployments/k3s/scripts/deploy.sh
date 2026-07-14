#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "${SCRIPT_DIR}/lib.sh"
load_config

WITH_OLLAMA=true
WITH_OPERATOR_AI=true
SKIP_REGISTRY_SECRET=false
SKIP_MONGO_SEED=false
OPERATOR_AI_ONLY=false

usage() {
  cat <<'EOF'
Usage: deployments/k3s/scripts/deploy.sh [options]

Deploy ReaperC2 on k3s with optional in-cluster Ollama for Operator AI.

Options:
  --no-ollama              Skip Ollama Deployment (use external Ollama URL in operator-ai.yaml)
  --no-operator-ai         Skip Operator AI ConfigMap/Secret
  --skip-registry-secret   Do not create registry pull secret
  --skip-mongo-seed        Do not run test/setup_mongo.sh after Mongo is ready
  --operator-ai-only       Apply Ollama + operator-ai.yaml and restart ReaperC2 only
  -h, --help               Show help

Prerequisites:
  - k3s with kubectl (or sudo k3s kubectl)
  - NFS export matching MONGO_NFS_* in k3s/config.env (or edit mongodb-nfs-pv.yaml)
  - k3s/mongo-secret.yaml credentials set
  - Image built/pushed: ./deployments/k3s/scripts/build-push-image.sh
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --no-ollama) WITH_OLLAMA=false; shift ;;
    --no-operator-ai) WITH_OPERATOR_AI=false; shift ;;
    --skip-registry-secret) SKIP_REGISTRY_SECRET=true; shift ;;
    --skip-mongo-seed) SKIP_MONGO_SEED=true; shift ;;
    --operator-ai-only) OPERATOR_AI_ONLY=true; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage; exit 1 ;;
  esac
done

if [[ "$OPERATOR_AI_ONLY" == true ]]; then
  if [[ "$WITH_OLLAMA" == true ]]; then
    apply_ollama_stack "$K8S_ROOT"
  fi
  if [[ "$WITH_OPERATOR_AI" == true ]]; then
    apply_operator_ai "$K8S_ROOT"
  fi
  rollout_reaperc2 "$REAPERC2_NAMESPACE"
  echo "Operator AI stack updated."
  exit 0
fi

require_file "${K3S_ROOT}/mongo-secret.yaml" "Edit deployments/k3s/mongo-secret.yaml before deploy"

if [[ "$SKIP_REGISTRY_SECRET" != true ]]; then
  if [[ -n "${REGISTRY_USERNAME:-}" && -n "${REGISTRY_PASSWORD:-}" ]]; then
    "${SCRIPT_DIR}/create-registry-secret.sh"
  else
    echo "Warning: REGISTRY_USERNAME/PASSWORD not set in k3s/config.env — ensure pull secret exists." >&2
  fi
fi

echo "Applying namespaces..."
kubectl_cmd apply -f "${K3S_ROOT}/namespace.yaml"

if [[ "$WITH_OLLAMA" == true ]]; then
  apply_ollama_stack "$K8S_ROOT"
fi

if [[ "$WITH_OPERATOR_AI" == true ]]; then
  apply_operator_ai "$K8S_ROOT"
fi

echo "Applying MongoDB..."
kubectl_cmd apply -f "${K3S_ROOT}/mongo-secret.yaml"
apply_k3s_mongo_nfs_pv
kubectl_cmd apply -f "${K3S_ROOT}/mongodb.yaml"

echo "Waiting for MongoDB PVC..."
kubectl_cmd -n reaperc2-ns wait --for=jsonpath='{.status.phase}'=Bound pvc/mongo-pvc --timeout=180s
kubectl_cmd -n reaperc2-ns rollout status statefulset/mongo --timeout=600s

if [[ "$SKIP_MONGO_SEED" != true ]]; then
  echo "Seeding api_db (test/setup_mongo.sh)..."
  "${SCRIPT_DIR}/seed-mongo.sh"
fi

echo "Applying ReaperC2..."
kubectl_cmd apply -f "${K3S_ROOT}/deployment.yaml"
kubectl_cmd apply -f "${K3S_ROOT}/service.yaml"
kubectl_cmd -n reaperc2-ns set image deployment/reaperc2-deployment \
  reaperc2="$(reaperc2_image_ref)"

kubectl_cmd -n reaperc2-ns rollout status deployment/reaperc2-deployment --timeout=300s

echo ""
echo "Deploy complete."
kubectl_cmd -n reaperc2-ns get pods,svc,pvc
if [[ "$WITH_OLLAMA" == true ]]; then
  kubectl_cmd -n ollama-ns get pods,svc,pvc
fi
echo ""
echo "Admin UI: kubectl port-forward -n reaperc2-ns deployment/reaperc2-deployment 8443:8443"
echo "Operator AI: open /ai after port-forward (Ollama provider should appear when Ollama pod is ready)."
