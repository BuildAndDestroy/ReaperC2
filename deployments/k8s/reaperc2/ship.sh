#!/usr/bin/env bash
# Build, push registry.reaper-ut.com :latest, restart pods so imagePullPolicy: Always pulls the new digest.
# Kubernetes does not watch :latest — a rollout is required after every push.
#
#   ./ship.sh                  # amd64, tag :latest, apply-core + restart
#   ./ship.sh --arch both      # multi-arch manifest at :latest
#   ./ship.sh --push-only      # registry only, no kubectl
#   ./ship.sh --deploy-only    # apply-core + restart, no rebuild
#
# ECR_REGISTRY defaults to registry.reaper-ut.com (same as Makefile / deployment.yaml).

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$HERE"

ARCH_RAW="${REAPER_IMAGE_ARCH:-amd64}"
PUSH_ONLY=0
DEPLOY_ONLY=0
MAKE_ARGS=()

usage() {
  cat <<'EOF'
Usage: ./ship.sh [options] [make-vars...]

  1. Build ReaperC2 and push to registry.reaper-ut.com as :latest (also aliases the git SHA).
  2. kubectl apply -k the overlay (image: registry.reaper-ut.com/reaperc2:latest) and rollout restart.

Options:
  --arch amd64|arm64|both   Image arch (default amd64). Aliases: x86_64, arm, multi.
  --push-only               Push registry only; do not apply or restart.
  --deploy-only             Apply overlay + rollout; do not build (image already in the registry).
  -h, --help                This help.

Environment:
  ECR_REGISTRY, AWS_REGION    Default registry.reaper-ut.com / us-east-1.
  IMAGE_TAG                   Default latest. Extra tags: git SHA, and :latest if you override IMAGE_TAG.
  AWS_CLI_PROFILE, SCYTHE_GIT_REF, REAPER_NS, REAPER_CLUSTER — same as make / deploy-cluster.sh.

Examples:
  cd deployments/k8s/reaperc2
  ./ship.sh
  ./ship.sh --push-only
  ./ship.sh --deploy-only
EOF
}

die() { echo "error: $*" >&2; exit 1; }

while [[ $# -gt 0 ]]; do
  case "$1" in
    --arch|-a)
      [[ $# -ge 2 ]] || die "--arch requires a value"
      ARCH_RAW="$2"
      shift 2
      ;;
    --push-only) PUSH_ONLY=1; shift ;;
    --deploy-only) DEPLOY_ONLY=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) MAKE_ARGS+=("$1"); shift ;;
  esac
done

export ECR_REGISTRY="${ECR_REGISTRY:-registry.reaper-ut.com}"
export AWS_REGION="${AWS_REGION:-us-east-1}"
export IMAGE_TAG="${IMAGE_TAG:-latest}"

if [[ "$PUSH_ONLY" -eq 1 && "$DEPLOY_ONLY" -eq 1 ]]; then
  die "use only one of --push-only or --deploy-only"
fi

if [[ "$DEPLOY_ONLY" -eq 1 ]]; then
  echo "==> Apply overlay and restart so pods pull :latest"
  ./reroll.sh --apply-core
  exit 0
fi

echo "==> Registry $ECR_REGISTRY  tag=$IMAGE_TAG  arch=$ARCH_RAW"

if [[ ${#MAKE_ARGS[@]} -gt 0 ]]; then
  ./build-push-image.sh --arch "$ARCH_RAW" "${MAKE_ARGS[@]}"
else
  ./build-push-image.sh --arch "$ARCH_RAW"
fi

if [[ "$PUSH_ONLY" -eq 1 ]]; then
  echo "==> Push complete (--push-only). Restart later with: ./reroll.sh --apply-core"
  exit 0
fi

echo "==> Apply overlay and restart so pods pull :latest"
./reroll.sh --apply-core
