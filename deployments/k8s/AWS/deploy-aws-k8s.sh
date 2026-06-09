#!/usr/bin/env bash
# Compatibility wrapper — canonical scripts live in ../reaperc2/
set -euo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export REAPER_CLUSTER="${REAPER_CLUSTER:-aws}"
exec "$HERE/../reaperc2/deploy-cluster.sh" "$@"
