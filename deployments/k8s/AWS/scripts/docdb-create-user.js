// Create the ReaperC2 application user (idempotent). Requires admin connection (authSource=admin).
const dbName = process.env.MONGO_DATABASE;
const appUser = process.env.MONGO_USERNAME;
const appPass = process.env.MONGO_PASSWORD;

if (!dbName || !appUser || !appPass) {
  print("error: MONGO_DATABASE, MONGO_USERNAME, and MONGO_PASSWORD must be set");
  quit(1);
}

const targetDb = db.getSiblingDB(dbName);
try {
  targetDb.createUser({
    user: appUser,
    pwd: appPass,
    roles: [{ role: "readWrite", db: dbName }],
  });
  print("Created user " + appUser);
} catch (e) {
  if (e.codeName === "DuplicateKey" || String(e).includes("already exists")) {
    print("User already exists: " + appUser);
  } else {
    throw e;
  }
}
