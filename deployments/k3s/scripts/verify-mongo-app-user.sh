#!/usr/bin/env bash
# Verify app user can authenticate (reads credentials from mongo-secret.yaml).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "${SCRIPT_DIR}/lib.sh"
load_config

MONGO_SECRET="${K3S_ROOT}/mongo-secret.yaml"
NS="${REAPERC2_NAMESPACE}"

read_mongo_app_creds() {
  python3 - <<'PY' "$MONGO_SECRET"
import sys
try:
    import yaml
except ImportError:
    print("PyYAML required: pip install pyyaml", file=sys.stderr)
    sys.exit(2)
data = yaml.safe_load(open(sys.argv[1]))
s = data["stringData"]
print(s["app_username"])
print(s["app_password"], end="")
PY
}

mapfile -t CREDS < <(read_mongo_app_creds)
APP_USER="${CREDS[0]}"
APP_PASS="${CREDS[1]}"
DB="${MONGO_DATABASE:-reaperc2-metric}"

echo "Verifying MongoDB app user ${APP_USER} on database ${DB}..."
if kubectl_cmd -n "$NS" exec statefulset/mongo -- mongosh --quiet \
  -u "$APP_USER" \
  -p "$APP_PASS" \
  --authenticationDatabase "$DB" \
  --eval 'db.runCommand({ ping: 1 }).ok'; then
  echo "MongoDB app authentication OK."
  exit 0
fi

echo "MongoDB app authentication failed." >&2
exit 1
