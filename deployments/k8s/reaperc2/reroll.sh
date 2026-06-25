#!/usr/bin/env bash
# Restart ReaperC2 pods and optionally refresh in-cluster config before restart.
# Typical uses: new image already in deployment.yaml / new ECR tag, rotated AI secrets, updated operator-ai.local.yaml.
#
#   ./reroll.sh                      # rollout restart + wait (same image/spec already in the cluster)
#   ./reroll.sh --apply-core         # kubectl apply -k overlay (picks up edited base/deployment.yaml, ConfigMaps, etc.), then restart
#   ./reroll.sh --apply-secrets      # kubectl apply *.local.yaml + operator-ai.local.yaml, then restart
#   ./reroll.sh --refresh-ecr        # refresh ECR docker-registry secret (same rules as deploy-cluster.sh), then restart
#   ./reroll.sh --apply-core --apply-secrets --refresh-ecr

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$HERE"

APPLY_SECRETS=0
REFRESH_ECR=0
APPLY_CORE=0

usage() {
  cat <<'EOF'
Usage: ./reroll.sh [options]

  Runs deploy-cluster.sh steps then rollout restart (pods pick up Secrets/ConfigMaps on restart).

Options:
  --apply-core      Apply the kustomize overlay (Deployment image, ConfigMaps, SA, Service, …). Use after you edit
                    base/deployment.yaml or other tracked YAML — reroll alone does NOT apply manifest changes from git.
  --apply-secrets   Apply namespace + documentdb/admin-bootstrap secrets + operator-ai.local.yaml (if present).
  --refresh-ecr   Recreate reaperc2-myregistrykey (requires aws + REAPER_CLUSTER=aws or REAPER_ECR_SECRET=1 on k3s).

Environment:
  REAPER_NS, REAPER_CLUSTER, KUBECTL, SKIP_ECR_SECRET, REAPER_ECR_SECRET — same as deploy-cluster.sh.

Examples:
  ./reroll.sh
  ./reroll.sh --apply-core                    # new image: line in base/deployment.yaml + this + build/push first
  kubectl apply -f examples/documentdb-secret.local.yaml && ./reroll.sh
  ./reroll.sh --apply-secrets
  ./reroll.sh --refresh-ecr
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --apply-core) APPLY_CORE=1; shift ;;
    --apply-secrets) APPLY_SECRETS=1; shift ;;
    --refresh-ecr) REFRESH_ECR=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "error: unknown option: $1" >&2; usage >&2; exit 1 ;;
  esac
done

if [[ "$APPLY_SECRETS" -eq 1 ]]; then
  ./deploy-cluster.sh apply-secrets
fi

if [[ "$REFRESH_ECR" -eq 1 ]]; then
  ./deploy-cluster.sh ecr-secret
fi

if [[ "$APPLY_CORE" -eq 1 ]]; then
  ./deploy-cluster.sh apply-core
fi

./deploy-cluster.sh rollout
