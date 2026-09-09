#!/usr/bin/env bash
# Build ReaperC2 container binaries and push to ECR (wraps repo Makefile).
# Run from anywhere; discovers the git repo root automatically.
#
#   ./build-push-image.sh --arch amd64     # linux/amd64 only (x86_64 clusters)
#   ./build-push-image.sh --arch arm64     # linux/arm64 only (ARM k3s / Graviton)
#   ./build-push-image.sh --arch both      # amd64 + arm64 + multi-arch manifest (default make build)
#
# Requires: Docker with buildx, Go, AWS CLI (same as `make build` / `make help`).
# Set ECR_REGISTRY, AWS_REGION, ECR_REPOSITORY, IMAGE_TAG, AWS_CLI_PROFILE, SCYTHE_GIT_REF as for make.
# IMAGE_TAG defaults to latest. Daily cluster update: ./ship.sh (push :latest + restart).
#
# Note: make build-amd64 / build-arm64 still run `build-binaries` (compiles linux/amd64 and linux/arm64
# Go binaries on the host); only the selected image is packaged and pushed.

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$HERE/../../.." && pwd)"

usage() {
  cat <<'EOF'
Usage: ./build-push-image.sh --arch <amd64|arm64|both> [make-vars...]

  Runs `make` from the ReaperC2 repo root to compile and push the image (default registry.reaper-ut.com).

  --arch amd64   Same as `make build-amd64` — push :TAG-amd64 and set the :TAG manifest to amd64-only.
  --arch arm64   Same as `make build-arm64` — push :TAG-arm64 and set the :TAG manifest to arm64-only.
  --arch both    Same as `make build` — push both arches and create a multi-arch manifest at :TAG.

  Aliases for --arch: x86_64→amd64, aarch64|arm→arm64, multi|all→both.

Environment (optional, passed to make):
  ECR_REGISTRY     default registry.reaper-ut.com
  AWS_REGION       e.g. us-east-1 (for get-login-password)
  ECR_REPOSITORY   default reaperc2
  IMAGE_TAG        default latest (also aliases git SHA; :latest is always pushed)
  AWS_CLI_PROFILE  SSO / named profile (see scripts/aws-for-make.sh)
  SCYTHE_GIT_REF   Scythe submodule ref for the image build

  Or set REAPER_IMAGE_ARCH=amd64|arm64|both instead of --arch.

Examples:
  cd deployments/k8s/reaperc2
  chmod +x build-push-image.sh ship.sh
  ./ship.sh                              # push :latest (amd64) and restart pods
  ./build-push-image.sh --arch amd64
  ./build-push-image.sh --arch arm64 AWS_CLI_PROFILE=my-sso IMAGE_TAG=v1.2.3
  REAPER_IMAGE_ARCH=both ./build-push-image.sh

After a successful push, Deployment image: .../reaperc2:latest (imagePullPolicy: Always).
Restart pods to pull: ./ship.sh (includes restart) or ./reroll.sh --apply-core
EOF
}

die() { echo "error: $*" >&2; exit 1; }

ARCH_RAW="${REAPER_IMAGE_ARCH:-}"
MAKE_ARGS=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --arch|-a)
      [[ $# -ge 2 ]] || die "--arch requires a value"
      ARCH_RAW="$2"
      shift 2
      ;;
    -h|--help) usage; exit 0 ;;
    *) MAKE_ARGS+=("$1"); shift ;;
  esac
done

[[ -n "$ARCH_RAW" ]] || die "specify --arch amd64|arm64|both (or set REAPER_IMAGE_ARCH)"

norm="$(printf '%s' "$ARCH_RAW" | tr '[:upper:]' '[:lower:]')"
case "$norm" in
  amd64|amd|x86_64) TARGET="build-amd64" ;;
  arm64|arm|aarch64) TARGET="build-arm64" ;;
  both|multi|all) TARGET="build" ;;
  *) die "unknown --arch '$ARCH_RAW' (use amd64, arm64, or both)" ;;
esac

[[ -f "$REPO_ROOT/Makefile" ]] || die "Makefile not found at $REPO_ROOT (expected ReaperC2 repo root)"
[[ -f "$REPO_ROOT/Dockerfile.pack" ]] || die "Dockerfile.pack not found at $REPO_ROOT"

echo "==> Repo: $REPO_ROOT"
if [[ ${#MAKE_ARGS[@]} -gt 0 ]]; then
  echo "==> make $TARGET ${MAKE_ARGS[*]}"
  (cd "$REPO_ROOT" && make "$TARGET" "${MAKE_ARGS[@]}")
else
  echo "==> make $TARGET"
  (cd "$REPO_ROOT" && make "$TARGET")
fi

echo ""
echo "Pushed ${ECR_REGISTRY:-registry.reaper-ut.com}/reaperc2:${IMAGE_TAG:-latest}"
echo "Cluster spec uses registry.reaper-ut.com/reaperc2:latest (imagePullPolicy: Always). Restart pods with:"
echo "  ./ship.sh            # this push + apply-core + rollout"
echo "  ./reroll.sh --apply-core"
