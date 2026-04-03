#!/usr/bin/env bash
# Local automation: MongoDB in Docker + mongoclient image runs setup_mongo.sh.
# Requires Docker. CI-friendly (no TTY required).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NET="${DOCKER_NETWORK:-reaperc2-test-net}"
MONGO_CTR="${MONGO_CONTAINER:-reaperc2-test-mongo}"

cleanup() {
  if [[ "${KEEP_MONGO:-0}" == "1" ]]; then
    echo "KEEP_MONGO=1: leaving ${MONGO_CTR} on network ${NET} (remove manually when done)."
    return
  fi
  docker rm -f "${MONGO_CTR}" >/dev/null 2>&1 || true
  if [[ "${KEEP_TEST_NETWORK:-0}" != "1" ]]; then
    docker network rm "${NET}" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

docker network inspect "${NET}" >/dev/null 2>&1 || docker network create "${NET}"

docker build -t mongoclient -f "${SCRIPT_DIR}/mongoclient.dockerfile" "${SCRIPT_DIR}"

docker run --rm -d \
  --name "${MONGO_CTR}" \
  --network "${NET}" \
  -p 27017:27017 \
  -e MONGO_INITDB_ROOT_USERNAME="${MONGO_ADMIN_USER:-admin}" \
  -e MONGO_INITDB_ROOT_PASSWORD="${MONGO_ADMIN_PASSWORD:-supersecretpasswordlol}" \
  mongodb/mongodb-community-server:latest

echo "Waiting for MongoDB to accept connections..."
ready=0
for _ in $(seq 1 60); do
  if docker exec "${MONGO_CTR}" mongosh --quiet --eval "db.runCommand({ ping: 1 })" >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 1
done
if [[ "${ready}" != "1" ]]; then
  echo "MongoDB did not become ready in time." >&2
  exit 1
fi

docker run --rm \
  --network "${NET}" \
  -e MONGO_HOST="${MONGO_CTR}" \
  -e MONGO_PORT=27017 \
  -e MONGO_ADMIN_USER="${MONGO_ADMIN_USER:-admin}" \
  -e MONGO_ADMIN_PASSWORD="${MONGO_ADMIN_PASSWORD:-supersecretpasswordlol}" \
  -e MONGO_API_USER="${MONGO_API_USER:-api_user}" \
  -e MONGO_API_PASSWORD="${MONGO_API_PASSWORD:-api_mongoApiPassword}" \
  -e IMPORT_DATA_JSON="${IMPORT_DATA_JSON:-1}" \
  mongoclient

echo "Mongo seed finished."
