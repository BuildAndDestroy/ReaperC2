#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT="${SCRIPT_DIR}/rds-combined-ca-bundle.pem"
URL="https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem"

echo "Downloading DocumentDB/RDS CA bundle to ${OUT}"
curl -fsSL -o "${OUT}" "${URL}"
echo "Done. Apply core stack from an overlay, e.g.:"
echo "  kubectl apply -k ${SCRIPT_DIR}/../overlays/aws-ecr"
echo "  kubectl apply -k ${SCRIPT_DIR}/../overlays/k3s"
