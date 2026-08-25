// Creates readWrite user on reaperc2-metric DB (runs once on empty data dir).
// Requires env: MONGO_APP_USERNAME, MONGO_APP_PASSWORD
const user = process.env.MONGO_APP_USERNAME;
const pwd = process.env.MONGO_APP_PASSWORD;
if (!user || !pwd) {
  throw new Error("MONGO_APP_USERNAME and MONGO_APP_PASSWORD must be set");
}

const dbName = "reaperc2-metric";
db = db.getSiblingDB(dbName);

if (db.getUser(user)) {
  db.updateUser(user, { pwd });
  print(`Updated password for app user ${user} on ${dbName}`);
  quit(0);
}

db.createUser({
  user,
  pwd,
  roles: [{ role: "readWrite", db: dbName }],
});

print(`Created app user ${user} on ${dbName}`);
