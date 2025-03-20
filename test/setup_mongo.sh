#!/bin/bash

MONGO_URI="mongodb://172.17.0.2:27017"
DB_NAME="api_db"
COLLECTION_CLIENTS="clients"
COLLECTION_HEARTBEAT="heartbeat"

echo "Creating MongoDB database and collection..."

# Test data, DO NOT USE IN PROD
# This will run as a test, containers will be killed and deleted on stop
mongosh "$MONGO_URI" <<EOF
use $DB_NAME;

db.createCollection("$COLLECTION_CLIENTS");
db.$COLLECTION_CLIENTS.createIndex({ "ClientId": 1 }, { unique: true });
db.$COLLECTION_CLIENTS.createIndex({ "Secret": 1 });
db.$COLLECTION_CLIENTS.insertMany([{"ClientId": "550e8400-e29b-41d4-a716-446655440000", "Secret": "mysecurekey1", "Active": true, "Connection_Type": "HTTP","Commands": ["ls -la", "whoami", "uptime"]},
{"ClientId": "660e9400-e29b-41d4-a716-556655440111", "Secret": "mysecurekey2", "Active": false, "Connection_Type": "TCP","Commands": []}
]);


db.createCollection("$COLLECTION_HEARTBEAT");
db.$COLLECTION_HEARTBEAT.insertOne({ "status": "ok" });

EOF

# Import JSON data
mongoimport --db "$DB_NAME" --collection data_json_as_collection --file data.json --jsonArray


echo "MongoDB setup complete!"

