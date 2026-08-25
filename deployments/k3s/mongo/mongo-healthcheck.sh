#!/bin/sh
# Probe script — password passed to mongosh as argv (safe for special characters).
set -e
mongosh --quiet \
  -u "${MONGO_INITDB_ROOT_USERNAME}" \
  -p "${MONGO_INITDB_ROOT_PASSWORD}" \
  --authenticationDatabase admin \
  --eval 'db.adminCommand("ping").ok'
