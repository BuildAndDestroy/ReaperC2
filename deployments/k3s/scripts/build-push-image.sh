#!/usr/bin/env bash
# Build linux/arm64 (Pi) or amd64 image and push to your registry.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "${SCRIPT_DIR}/lib.sh"
load_config

ARCH="${ARCH:-arm64}"
IMPORT_LOCAL=false
PUSH=true

usage() {
  cat <<'EOF'
Usage: deployments/k3s/scripts/build-push-image.sh [options]

Options:
  --arch arm64|amd64   Target platform (default: arm64 for Raspberry Pi k3s)
  --tag TAG            Image tag (default: from config.env REAPERC2_IMAGE_TAG)
  --import-local       Also import tarball into local k3s (offline fallback)
  --load               Build for local docker only (no push; single platform)
  -h, --help           Show help

Requires REGISTRY_* and REAPERC2_IMAGE in k3s/config.env (unless --load).
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --arch) ARCH="$2"; shift 2 ;;
    --tag) REAPERC2_IMAGE_TAG="$2"; shift 2 ;;
    --import-local) IMPORT_LOCAL=true; shift ;;
    --load) PUSH=false; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage; exit 1 ;;
  esac
done

IMAGE="$(reaperc2_image_ref)"
PLATFORM="linux/${ARCH}"

if [[ "$PUSH" == true ]]; then
  require_registry_credentials
  docker_build_push "$REPO_ROOT" "$IMAGE" "$PLATFORM" "$IMPORT_LOCAL"
  echo "Pushed ${IMAGE}"
else
  echo "Building ${IMAGE} for ${PLATFORM} (load locally)..."
  DOCKER_BUILD_LOAD=1 docker_build_push "$REPO_ROOT" "$IMAGE" "$PLATFORM" "$IMPORT_LOCAL"
  echo "Loaded ${IMAGE} into local Docker"
fi
