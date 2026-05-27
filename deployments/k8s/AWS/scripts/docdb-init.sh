#!/usr/bin/env bash
# Connect to DocumentDB with TLS and run scripts/docdb-init.js (idempotent).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=docdb-common.sh
source "${SCRIPT_DIR}/docdb-common.sh"

: "${MONGO_HOST:?MONGO_HOST required}"
: "${MONGO_PORT:?MONGO_PORT required}"
: "${MONGO_USERNAME:?MONGO_USERNAME required}"
: "${MONGO_PASSWORD:?MONGO_PASSWORD required}"
: "${MONGO_DATABASE:?MONGO_DATABASE required}"

CA_FILE="${MONGO_TLS_CA_FILE:-/certs/rds-combined-ca-bundle.pem}"
if [[ ! -f "${CA_FILE}" ]]; then
  echo "error: TLS CA bundle not found at ${CA_FILE}" >&2
  exit 1
fi

AUTH_DB="${MONGO_AUTH_SOURCE:-${MONGO_DATABASE}}"
URI="$(build_docdb_uri "/${MONGO_DATABASE}")"

echo "Connecting to ${MONGO_HOST}:${MONGO_PORT}/${MONGO_DATABASE} (TLS, authSource=${AUTH_DB}, SCRAM-SHA-1)"
export MONGO_DATABASE
mongosh "${URI}" \
  "${MONGOSH_AUTH_MECH[@]}" \
  --username "${MONGO_USERNAME}" \
  --password "${MONGO_PASSWORD}" \
  --authenticationDatabase "${AUTH_DB}" \
  --file /scripts/docdb-init.js
