#!/usr/bin/env bash
# Create the ReaperC2 application user on DocumentDB (run once with master/admin creds).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=docdb-common.sh
source "${SCRIPT_DIR}/docdb-common.sh"

: "${MONGO_HOST:?}"
: "${MONGO_PORT:?}"
: "${MONGO_ADMIN_USER:?}"
: "${MONGO_ADMIN_PASSWORD:?}"
: "${MONGO_USERNAME:?}"
: "${MONGO_PASSWORD:?}"
: "${MONGO_DATABASE:?}"

AUTH_DB="${MONGO_AUTH_SOURCE:-admin}"
URI="$(build_docdb_uri "/")"

echo "Creating user ${MONGO_USERNAME} on database ${MONGO_DATABASE} (authSource=${AUTH_DB})"
export MONGO_DATABASE MONGO_USERNAME MONGO_PASSWORD
mongosh "${URI}" \
  "${MONGOSH_AUTH_MECH[@]}" \
  --username "${MONGO_ADMIN_USER}" \
  --password "${MONGO_ADMIN_PASSWORD}" \
  --authenticationDatabase "${AUTH_DB}" \
  --file /scripts/docdb-create-user.js
