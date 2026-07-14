#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "${SCRIPT_DIR}/lib.sh"
load_config
require_registry_credentials

create_registry_pull_secret \
  "$REAPERC2_NAMESPACE" \
  "$REGISTRY_SECRET_NAME" \
  "${K3S_ROOT}/namespace.yaml"

echo "Registry pull secret ${REGISTRY_SECRET_NAME} applied in ${REAPERC2_NAMESPACE}."
