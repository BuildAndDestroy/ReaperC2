// Idempotent ReaperC2 DocumentDB schema init (collections + indexes).
// Run via docdb-init-job.sh after the app user exists.

const dbName = process.env.MONGO_DATABASE || "api_db";
const db = db.getSiblingDB(dbName);

function ensureCollection(name) {
  const names = db.getCollectionNames();
  if (names.indexOf(name) === -1) {
    db.createCollection(name);
    print("Created collection: " + name);
  } else {
    print("Collection exists: " + name);
  }
}

function ensureIndex(coll, keys, options) {
  const name = options && options.name ? options.name : JSON.stringify(keys);
  try {
    db.getCollection(coll).createIndex(keys, options || {});
    print("Index ok on " + coll + ": " + name);
  } catch (e) {
    if (e.codeName === "IndexOptionsConflict" || e.code === 85 || e.code === 86) {
      print("Index already exists on " + coll + ": " + name);
    } else {
      throw e;
    }
  }
}

print("ReaperC2 DocumentDB init for database: " + dbName);

ensureCollection("clients");
ensureIndex("clients", { ClientId: 1 }, { unique: true, name: "idx_clients_client_id" });
ensureIndex("clients", { Secret: 1 }, { name: "idx_clients_secret" });

ensureCollection("heartbeat");
ensureCollection("data");
ensureIndex("data", { ClientId: 1 }, { name: "idx_data_client_id" });

ensureCollection("operators");
ensureIndex("operators", { username: 1 }, { unique: true, name: "idx_operators_username" });

ensureCollection("operator_sessions");
ensureCollection("operator_mfa_challenges");
ensureIndex(
  "operator_mfa_challenges",
  { expires_at: 1 },
  { name: "idx_mfa_expires_at", expireAfterSeconds: 600 }
);

ensureCollection("engagements");
ensureIndex("engagements", { created_at: -1 }, { name: "idx_engagements_created_at" });
ensureIndex("engagements", { assigned_operators: 1 }, { name: "idx_engagements_assigned_operators" });

ensureCollection("audit_logs");
ensureIndex("audit_logs", { time: -1 }, { name: "idx_audit_time" });
ensureIndex(
  "audit_logs",
  { engagement_id: 1, time: -1 },
  { name: "idx_audit_engagement_time" }
);

ensureCollection("file_artifacts");
ensureIndex(
  "file_artifacts",
  { client_id: 1, created_at: -1 },
  { name: "idx_file_artifacts_client_created" }
);

ensureCollection("beacon_profiles");
ensureIndex("beacon_profiles", { created_at: -1 }, { name: "idx_beacon_profiles_created_at" });

ensureCollection("operator_chat");
ensureIndex("operator_chat", { created_at: -1 }, { name: "idx_operator_chat_created_at" });

print("DocumentDB init complete.");
