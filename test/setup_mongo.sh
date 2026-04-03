#!/usr/bin/env bash
set -euo pipefail

ADMINUSER="${MONGO_ADMIN_USER:-admin}"
ADMINPASSWORD="${MONGO_ADMIN_PASSWORD:-supersecretpasswordlol}"
APIUSER="${MONGO_API_USER:-api_user}"
APIUSERPASSWORD="${MONGO_API_PASSWORD:-api_mongoApiPassword}"

# Override for local Docker (e.g. run_tests.sh) or in-cluster exec:
#   MONGO_HOST=mongodb MONGO_PORT=27017 ./setup_mongo.sh
MONGO_HOST="${MONGO_HOST:-mongodb-service.reaperc2-ns.svc.cluster.local}"
MONGO_PORT="${MONGO_PORT:-27017}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DATA_JSON="${DATA_JSON:-$SCRIPT_DIR/data.json}"
# Set to 0 to skip optional mongoimport of data.json
IMPORT_DATA_JSON="${IMPORT_DATA_JSON:-1}"
DATA_JSON_COLLECTION="${DATA_JSON_COLLECTION:-seed_docs}"

MONGO_URI="mongodb://${ADMINUSER}:${ADMINPASSWORD}@${MONGO_HOST}:${MONGO_PORT}"
DB_API_NAME="api_db"
DB_DATA_COLLECTION="data"
COLLECTION_CLIENTS="clients"
COLLECTION_HEARTBEAT="heartbeat"

echo "Connecting to ${MONGO_HOST}:${MONGO_PORT}"
echo "Creating MongoDB database $DB_API_NAME and collections: $COLLECTION_CLIENTS, $COLLECTION_HEARTBEAT, $DB_DATA_COLLECTION..."

# Test data, DO NOT USE IN PROD
# This will run as a test, containers will be killed and deleted on stop
mongosh "$MONGO_URI" <<EOF
use $DB_API_NAME;

db.createCollection("$COLLECTION_CLIENTS");
db.$COLLECTION_CLIENTS.createIndex({ "ClientId": 1 }, { unique: true });
db.$COLLECTION_CLIENTS.createIndex({ "Secret": 1 });
db.$COLLECTION_CLIENTS.insertMany([
    {
        "ClientId": "550e8400-e29b-41d4-a716-446655440000",
        "Secret": "mysecurekey1",
        "ExpectedHeartBeat": "5s",
        "Active": true,
        "Connection_Type": "HTTP",
        "Commands": ["ls -alh", "whoami", "uptime", "date", "groups", "uname -a", "which kubectl", "df -h", "docker images"]
    },
    {
        "ClientId": "660e9400-e29b-41d4-a716-556655440111",
        "Secret": "mysecurekey2",
        "ExpectedHeartBeat": "30s",
        "Active": false,
        "Connection_Type": "TCP",
        "Pivot Server": "",
        "Commands": []
    },
    {
        "ClientId": "17b7c8aa-a781-41c7-93d8-96a3f37c97ce",
        "Secret": "mysecurekey3",
        "ExpectedHeartBeat": "5s",
        "Active": true,
        "Connection_Type": "HTTP",
        "Commands": ["ls -alh", "whoami", "uptime", "date", "groups", "uname -a", "which kubectl", "df -h", "docker images"]
    },
    {
        "ClientId": "3187f868-1daf-4713-b91d-40cbed639b6e",
        "Secret": "mysecurekey4",
        "ExpectedHeartBeat": "10s",
        "Active": true,
        "Connection_Type": "HTTP",
        "Commands": ["dir", "whoami /all", "(Get-CimInstance -ClassName Win32_OperatingSystem).LastBootUpTime", "date", "net user", "net user /domain", "net localgroup"]
    },
    {
        "ClientId": "f51fceaf-67b7-4428-9a6f-f7dba90f9f2b",
        "Secret": "mysecurekey5",
        "ExpectedHeartBeat": "5s",
        "Active": true,
        "Connection_Type": "HTTP",
        "Commands": ["dir", "whoami /all", "date", "net user", "net localgroup"]
    }
]);


db.createCollection("$COLLECTION_HEARTBEAT");
db.$COLLECTION_HEARTBEAT.insertOne({ "status": "ok" });


db.createCollection("$DB_DATA_COLLECTION");

db.$DB_DATA_COLLECTION.createIndex({ "ClientId": 1 });

db.$DB_DATA_COLLECTION.insertMany([
    {
        "ClientId": "550e8400-e29b-41d4-a716-446655440000",
        "info": "Sample data for client 1",
        "user": "",
        "hostname": "",
        "ip_address": "",
        "groups": [],
        "createdAt": new Date()
    },
    {
        "ClientId": "660e9400-e29b-41d4-a716-556655440111",
        "info": "Sample data for client 2",
        "user": "",
        "hostname": "",
        "ip_address": "",
        "groups": [],
        "createdAt": new Date()
    }
]);
EOF

# Test data - 
# mongosh "$MONGO_URI" --authenticationDatabase admin <<EOF
mongosh "$MONGO_URI" <<EOF
use $DB_API_NAME;
db.createUser({
  user: "$APIUSER",
  pwd: "$APIUSERPASSWORD",
  roles: [
    { role: "readWrite", db: "$DB_API_NAME" }
  ]
});

print("✅ Created $APIUSER with access to $DB_API_NAME");
EOF

if [[ "${IMPORT_DATA_JSON}" == "1" ]] && [[ -f "${DATA_JSON}" ]]; then
  echo "Importing ${DATA_JSON} -> ${DB_API_NAME}.${DATA_JSON_COLLECTION}"
  mongoimport \
    --uri="${MONGO_URI}/${DB_API_NAME}?authSource=admin" \
    --collection="${DATA_JSON_COLLECTION}" \
    --file="${DATA_JSON}" \
    --jsonArray
else
  echo "Skipping data.json import (IMPORT_DATA_JSON=${IMPORT_DATA_JSON} or missing file)."
fi

echo "MongoDB setup complete!"

