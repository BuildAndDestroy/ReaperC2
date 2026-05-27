# Shared DocumentDB TLS URI + auth for mongosh jobs (source from other scripts).
# DocumentDB returns "Unsupported mechanism [ -301 ]" if mongosh negotiates SCRAM-SHA-256 only.
build_docdb_uri() {
  local db_path="${1:-/}"
  local ca="${MONGO_TLS_CA_FILE:-/certs/rds-combined-ca-bundle.pem}"
  local mech="${MONGO_AUTH_MECHANISM:-SCRAM-SHA-1}"
  echo "mongodb://${MONGO_HOST}:${MONGO_PORT}${db_path}?tls=true&tlsCAFile=${ca}&replicaSet=rs0&readPreference=secondaryPreferred&retryWrites=false&authMechanism=${mech}"
}

# shellcheck disable=SC2034
MONGOSH_AUTH_MECH=(--authenticationMechanism "${MONGO_AUTH_MECHANISM:-SCRAM-SHA-1}")
