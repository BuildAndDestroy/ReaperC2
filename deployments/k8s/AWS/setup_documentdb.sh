#!/bin/bash

ADMINUSER="docdbadmin"
ADMINPASSWORD='Z1Th!s!i55Om3S3crEt123!'
APIUSER="api_user"
APIUSERPASSWORD="api_mongoApiPassword"
# MONGOIPADDR="172.17.0.2"
MONGOIPADDR="prodsec-docdb-cluster.cluster-chgnk9set74z.us-east-1.docdb.amazonaws.com"
MONGOPORT="27017"
DOCUMENTDBCONNECTIONSTRING="?tls=true&tlsCAFile=/certs/rds-combined-ca-bundle.pem&replicaSet=rs0&readPreference=secondaryPreferred&retryWrites=false"
MONGO_URI="mongodb://$ADMINUSER:$ADMINPASSWORD@$MONGOIPADDR:$MONGOPORT/$DOCUMENTDBCONNECTIONSTRING"
DB_API_NAME="api_db"
DB_DATA_COLLECTION="data"
COLLECTION_CLIENTS="clients"
COLLECTION_HEARTBEAT="heartbeat"

echo "Creating MongoDB database $DB_API_NAME and collections: $COLLECTION_CLIENTS, $COLLECTION_HEARTBEAT, and $COLLECTION_CLIENTS..."

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
        groups: "",
        "hostname": "",
        "ip_address": "",
        "groups": [],
        "createdAt": new Date()
    },
    {
        "ClientId": "660e9400-e29b-41d4-a716-556655440111",
        "info": "Sample data for client 2",
        "user": "",
        groups: "",
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

# Import JSON data
#mongoimport --db "$DB_API_NAME" --collection data_json_as_collection --file data.json --jsonArray


echo "MongoDB setup complete!"
