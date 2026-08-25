#!/usr/bin/env bash
# Rebuild image, redeploy ReaperC2 on k3s, optionally refresh Operator AI stack.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "${SCRIPT_DIR}/lib.sh"
load_config

WITH_OPERATOR_AI=false
WITH_INGRESS=false
REFRESH_INGRESS=false
FRESH_MONGO=false
BUILD_ARGS=()

usage() {
  cat <<'EOF'
Usage: deployments/k3s/scripts/reroll.sh [options]

Rebuild/push image and roll out ReaperC2 (and optionally Operator AI).

Options:
  --with-operator-ai     Apply Ollama + operator-ai.yaml before restart
  --with-ingress         Pass --with-ingress to deploy on --fresh-mongo
  --refresh-ingress      Tear down TLS/ingress and re-apply (e.g. staging → prod)
  --fresh-mongo          Undeploy with PVC delete and full redeploy (destructive)
  -h, --help             Show help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --with-operator-ai) WITH_OPERATOR_AI=true; shift ;;
    --with-ingress) WITH_INGRESS=true; shift ;;
    --refresh-ingress) REFRESH_INGRESS=true; shift ;;
    --fresh-mongo) FRESH_MONGO=true; shift ;;
    --build-arg)
      BUILD_ARGS+=("$2")
      shift 2
      ;;
    -h|--help) usage; exit 0 ;;
    *) BUILD_ARGS+=("$1"); shift ;;
  esac
done

if [[ "$FRESH_MONGO" == true ]]; then
  "${SCRIPT_DIR}/undeploy.sh" --delete-pvc
  echo ""
  echo "Clear the NFS export (MONGO_NFS_PATH in k3s/config.env) before continuing."
  read -r -p "NFS folder cleared? [y/N] " confirm
  if [[ "${confirm,,}" != "y" ]]; then
    echo "Aborted."
    exit 1
  fi
  deploy_args=()
  [[ "$WITH_INGRESS" == true ]] && deploy_args+=(--with-ingress)
  "${SCRIPT_DIR}/deploy.sh" "${deploy_args[@]}"
  exit 0
fi

if [[ "$REFRESH_INGRESS" == true ]]; then
  teardown_k3s_ingress "$K3S_ROOT"
  apply_k3s_ingress "$K3S_ROOT"
  echo "Ingress refreshed (issuer: ${CERT_MANAGER_ISSUER})."
  kubectl_cmd -n "$REAPERC2_NAMESPACE" get ingress,ingressroute,certificate 2>/dev/null || true
  exit 0
fi

"${SCRIPT_DIR}/build-push-image.sh" "${BUILD_ARGS[@]}"

kubectl_cmd -n "$REAPERC2_NAMESPACE" set image deployment/reaperc2-deployment \
  reaperc2="$(reaperc2_image_ref)"

if [[ "$WITH_OPERATOR_AI" == true ]]; then
  apply_ollama_stack "$K8S_ROOT" "$K3S_ROOT"
  apply_operator_ai "$K8S_ROOT"
fi

kubectl_cmd -n "$REAPERC2_NAMESPACE" rollout restart statefulset/mongo || true
kubectl_cmd -n "$REAPERC2_NAMESPACE" rollout status statefulset/mongo --timeout=600s || true
rollout_reaperc2 "$REAPERC2_NAMESPACE"

echo "Reroll complete."
kubectl_cmd -n "$REAPERC2_NAMESPACE" get pods
