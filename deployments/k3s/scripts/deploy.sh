#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "${SCRIPT_DIR}/lib.sh"
load_config

WITH_OLLAMA=false
WITH_OPERATOR_AI=true
WITH_INGRESS=false
SKIP_REGISTRY_SECRET=false
SKIP_MONGO_SEED=false
OPERATOR_AI_ONLY=false
OLLAMA_ONLY=false

usage() {
  cat <<'EOF'
Usage: deployments/k3s/scripts/deploy.sh [options]

Deploy ReaperC2 on k3s with optional in-cluster Ollama for Operator AI.

Ollama is **off by default** (`--no-ollama` is the default). Use cloud providers in
operator-ai.yaml instead, or pass explicit flags to enable Ollama if you add it back.

Slow rollout: deploy Ollama first (model pulls can take 30+ minutes on Pi):
  ./deployments/k3s/scripts/deploy.sh --ollama-only

Beacon traffic: use --with-ingress for public HTTPS on INGRESS_HOST (default
metrics.harvestrangelabs.com → service :8080). Admin UI is never on ingress;
use kubectl port-forward :8443 (same as AWS reaperc2 path).

Options:
  --ollama-only            Apply Ollama only (NFS + model pull); skip ReaperC2/Mongo
  --with-ollama            Apply in-cluster Ollama with full deploy
  --with-ingress           Apply Traefik ingress + IngressRoute for beacons
  --no-ollama              Skip Ollama Deployment (default)
  --no-operator-ai         Skip Operator AI ConfigMap/Secret
  --skip-registry-secret   Do not create registry pull secret
  --skip-mongo-seed        Do not run test/setup_mongo.sh after Mongo is ready
  --operator-ai-only       Apply operator-ai.yaml and restart ReaperC2 only (add --with-ollama to include Ollama)
  -h, --help               Show help

Prerequisites:
  - k3s with kubectl (or sudo k3s kubectl)
  - Empty NFS export matching MONGO_NFS_* in k3s/config.env (first install)
  - k3s/mongo-secret.yaml and admin-bootstrap-secret.yaml credentials set
  - Image built/pushed: ./deployments/k3s/scripts/build-push-image.sh
  - For --with-ingress: Traefik + cert-manager ClusterIssuer (CERT_MANAGER_ISSUER)
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --with-ingress) WITH_INGRESS=true; shift ;;
    --with-ollama) WITH_OLLAMA=true; shift ;;
    --no-ollama) WITH_OLLAMA=false; shift ;;
    --no-operator-ai) WITH_OPERATOR_AI=false; shift ;;
    --skip-registry-secret) SKIP_REGISTRY_SECRET=true; shift ;;
    --skip-mongo-seed) SKIP_MONGO_SEED=true; shift ;;
    --operator-ai-only) OPERATOR_AI_ONLY=true; shift ;;
    --ollama-only) OLLAMA_ONLY=true; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage; exit 1 ;;
  esac
done

if [[ "$OLLAMA_ONLY" == true ]]; then
  echo "Ollama-only deploy (models: ${OLLAMA_PULL_MODELS:-from config.env})..."
  echo "Init pull can take a long time on Pi — watch: kubectl logs -n ollama-ns -l app=ollama -c pull-models -f"
  apply_ollama_stack "$K8S_ROOT" "$K3S_ROOT"
  echo ""
  echo "Ollama stack applied."
  kubectl_cmd -n ollama-ns get pods,svc,pvc,pv 2>/dev/null || true
  exit 0
fi

if [[ "$OPERATOR_AI_ONLY" == true ]]; then
  if [[ "$WITH_OLLAMA" == true ]]; then
    apply_ollama_stack "$K8S_ROOT" "$K3S_ROOT"
  fi
  if [[ "$WITH_OPERATOR_AI" == true ]]; then
    apply_operator_ai "$K8S_ROOT"
  fi
  rollout_reaperc2 "$REAPERC2_NAMESPACE"
  echo "Operator AI stack updated."
  exit 0
fi

require_file "${K3S_ROOT}/mongo-secret.yaml" "Edit deployments/k3s/mongo-secret.yaml before deploy"
require_file "${K3S_ROOT}/admin-bootstrap-secret.yaml" "Edit deployments/k3s/admin-bootstrap-secret.yaml before deploy"

python3 "${SCRIPT_DIR}/verify-mongo-secret.py" --mongo-secret "${K3S_ROOT}/mongo-secret.yaml"

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
  apply_ollama_stack "$K8S_ROOT" "$K3S_ROOT"
fi

if [[ "$WITH_OPERATOR_AI" == true ]]; then
  apply_operator_ai "$K8S_ROOT"
fi

echo "Applying MongoDB..."
kubectl_cmd apply -f "${K3S_ROOT}/mongo-secret.yaml"
apply_k3s_mongo_nfs_pv
apply_k3s_mongo_configmaps
apply_k3s_mongodb

echo "Waiting for MongoDB PVC..."
kubectl_cmd -n "$REAPERC2_NAMESPACE" wait --for=jsonpath='{.status.phase}'=Bound pvc/mongo-pvc --timeout=180s
echo "Waiting for MongoDB pod (first empty NFS volume runs init: root + app user)..."
kubectl_cmd -n "$REAPERC2_NAMESPACE" rollout status statefulset/mongo --timeout=600s

echo "Verifying app user can authenticate..."
if ! "${SCRIPT_DIR}/verify-mongo-app-user.sh"; then
  echo "Retrying app user bootstrap on mongo pod..."
  "${SCRIPT_DIR}/mongo-bootstrap-app-user.sh"
  "${SCRIPT_DIR}/verify-mongo-app-user.sh"
fi

if [[ "$SKIP_MONGO_SEED" != true ]]; then
  echo "Seeding ${MONGO_DATABASE} (test/setup_mongo.sh)..."
  "${SCRIPT_DIR}/seed-mongo.sh"
fi

echo "Applying ReaperC2..."
kubectl_cmd apply -f "${K3S_ROOT}/admin-bootstrap-secret.yaml"
kubectl_cmd apply -f "${K3S_ROOT}/deployment.yaml"
kubectl_cmd apply -f "${K3S_ROOT}/service.yaml"
kubectl_cmd -n "$REAPERC2_NAMESPACE" set image deployment/reaperc2-deployment \
  reaperc2="$(reaperc2_image_ref)"

kubectl_cmd -n "$REAPERC2_NAMESPACE" rollout status deployment/reaperc2-deployment --timeout=300s

if [[ "$WITH_INGRESS" == true ]]; then
  apply_k3s_ingress "$K3S_ROOT"
fi

echo ""
echo "Deploy complete."
kubectl_cmd -n "$REAPERC2_NAMESPACE" get pods,svc,pvc
if [[ "$WITH_OLLAMA" == true ]]; then
  kubectl_cmd -n ollama-ns get pods,svc,pvc
fi
echo ""
if [[ "$WITH_INGRESS" == true ]]; then
  echo "Beacon: https://${INGRESS_HOST}/"
fi
echo "Admin UI (port-forward only): kubectl port-forward -n ${REAPERC2_NAMESPACE} deployment/reaperc2-deployment 8443:8443"
echo "  Default login (first boot): admin / changeme — edit admin-bootstrap-secret.yaml before deploy to change."
echo "Operator AI: open /ai after port-forward (Ollama provider should appear when Ollama pod is ready)."
