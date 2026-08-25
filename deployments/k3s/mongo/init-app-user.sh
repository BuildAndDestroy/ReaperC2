#!/bin/bash
# Runs after root user is created (mongo:7 entrypoint, first init only).
set -euo pipefail

mongosh -u "${MONGO_INITDB_ROOT_USERNAME}" -p "${MONGO_INITDB_ROOT_PASSWORD}" \
  --authenticationDatabase admin \
  /docker-entrypoint-initdb.d/init-app-user.js
