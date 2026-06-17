#!/usr/bin/env bash
# Convenience wrapper around deploy-cluster.sh with optional egress NetworkPolicy.
# Run from repo anywhere:  deployments/k8s/reaperc2/deploy.sh [options] <deploy-cluster command> [args]
#
#   ./deploy.sh --no-egress all          # full install without restricting pod egress (default behavior)
#   ./deploy.sh --with-egress all        # same as "all", then apply examples/networkpolicy-egress-restricted.local.yaml
#   ./deploy.sh apply-core               # core only, no NetworkPolicy change
#   ./deploy.sh --with-egress apply-core # apply manifest + egress policy (requires .local netpol file)
#
# Egress policy file: copy examples/networkpolicy-egress-restricted.yaml →
#   examples/networkpolicy-egress-restricted.local.yaml and set the DocumentDB CIDR.

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$HERE"

REAPER_NS="${REAPER_NS:-reaperc2-ns}"
NP_NAME="reaperc2-egress-restricted"
NP_FILE_LOCAL="$HERE/examples/networkpolicy-egress-restricted.local.yaml"

die() { echo "error: $*" >&2; exit 1; }
info() { echo "==> $*"; }

usage() {
  cat <<'EOF'
Usage: ./deploy.sh [options] <command> [args to deploy-cluster.sh]

  Thin wrapper around ./deploy-cluster.sh for ReaperC2 on Kubernetes (see README.md).

Options:
  --with-egress   After a successful "all" or "apply-core", apply
                  examples/networkpolicy-egress-restricted.local.yaml (you must copy from the template and edit CIDRs).
  --no-egress     Before "all" or "apply-core", delete NetworkPolicy reaperc2-egress-restricted (open pod egress).
                  Ignored for other commands except teardown always removes the policy after cluster teardown.

Commands:
  Same as ./deploy-cluster.sh — run ./deploy-cluster.sh help for the full list (all, apply-core, apply-ingress, …).

Environment:
  REAPER_NS, REAPER_CLUSTER, KUBECTL — passed through to deploy-cluster.sh where applicable.

Examples:
  chmod +x deploy.sh reroll.sh deploy-cluster.sh base/fetch-docdb-ca-bundle.sh
  ./deploy.sh check-local
  ./deploy.sh --no-egress all
  REAPER_CLUSTER=k3s ./deploy.sh all
  ./deploy.sh --with-egress all
  ./deploy.sh apply-ingress
EOF
}

WITH_EGRESS=0
NO_EGRESS=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    --with-egress) WITH_EGRESS=1; shift ;;
    --no-egress) NO_EGRESS=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) break ;;
  esac
done

if [[ $# -lt 1 ]]; then
  usage
  exit 0
fi

primary="$1"

if [[ "$NO_EGRESS" -eq 1 && ( "$primary" == "all" || "$primary" == "apply-core" ) ]]; then
  info "Removing NetworkPolicy $NP_NAME (open egress for this deploy)"
  "${KUBECTL:-kubectl}" delete networkpolicy "$NP_NAME" -n "$REAPER_NS" --ignore-not-found
fi

if ! ./deploy-cluster.sh "$@"; then
  exit 1
fi

if [[ "$primary" == "teardown" ]]; then
  info "Removing NetworkPolicy $NP_NAME (if present)"
  "${KUBECTL:-kubectl}" delete networkpolicy "$NP_NAME" -n "$REAPER_NS" --ignore-not-found
fi

if [[ "$WITH_EGRESS" -eq 1 && ( "$primary" == "all" || "$primary" == "apply-core" ) ]]; then
  [[ -f "$NP_FILE_LOCAL" ]] || die "missing $NP_FILE_LOCAL — cp examples/networkpolicy-egress-restricted.yaml examples/networkpolicy-egress-restricted.local.yaml && edit DocumentDB CIDR"
  info "Applying egress NetworkPolicy from $(basename "$NP_FILE_LOCAL")"
  "${KUBECTL:-kubectl}" apply -f "$NP_FILE_LOCAL"
  echo ""
  echo "Confirm your cluster CNI enforces NetworkPolicy. If pods cannot reach DocumentDB or HTTPS APIs, widen or fix ipBlock rules in $NP_FILE_LOCAL, then kubectl apply -f that file again."
fi
