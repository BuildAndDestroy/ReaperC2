#!/usr/bin/env bash
# Run first-boot app user script on mongo pod (idempotent). Used by deploy.sh if verify fails.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "${SCRIPT_DIR}/lib.sh"
load_config

NS="${REAPERC2_NAMESPACE}"

echo "Bootstrapping app user from reaperc2-mongodb-secrets on pod..."
kubectl_cmd -n "$NS" exec statefulset/mongo -- /docker-entrypoint-initdb.d/init-app-user.sh
