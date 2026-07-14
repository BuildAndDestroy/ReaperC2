#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "${SCRIPT_DIR}/lib.sh"
load_config

MONGO_SECRET="${K3S_ROOT}/mongo-secret.yaml"
require_file "$MONGO_SECRET" "Edit deployments/k3s/mongo-secret.yaml"

root_user="$(kubectl_cmd get secret reaperc2-mongodb-secrets -n reaperc2-ns -o jsonpath='{.data.root_username}' | base64 -d)"
root_pass="$(kubectl_cmd get secret reaperc2-mongodb-secrets -n reaperc2-ns -o jsonpath='{.data.root_password}' | base64 -d)"
app_user="$(kubectl_cmd get secret reaperc2-mongodb-secrets -n reaperc2-ns -o jsonpath='{.data.app_username}' | base64 -d)"
app_pass="$(kubectl_cmd get secret reaperc2-mongodb-secrets -n reaperc2-ns -o jsonpath='{.data.app_password}' | base64 -d)"

export MONGO_ADMIN_USER="$root_user"
export MONGO_ADMIN_PASSWORD="$root_pass"
export MONGO_API_USER="$app_user"
export MONGO_API_PASSWORD="$app_pass"
export MONGO_HOST="127.0.0.1"
export MONGO_PORT="27017"
export IMPORT_DATA_JSON="${IMPORT_DATA_JSON:-0}"

echo "Port-forwarding MongoDB..."
kubectl_cmd -n reaperc2-ns port-forward svc/mongodb-service 27017:27017 &
pf_pid=$!
trap 'kill "$pf_pid" 2>/dev/null || true' EXIT
sleep 2

"${REPO_ROOT}/test/setup_mongo.sh"
echo "Mongo seed complete."
