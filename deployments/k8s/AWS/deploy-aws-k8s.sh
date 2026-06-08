#!/usr/bin/env bash
# ReaperC2 AWS EKS — install / update helper (DocumentDB bundle).
# Run from this directory:  cd deployments/k8s/AWS && ./deploy-aws-k8s.sh help
#
# Design: core stack (kustomize) excludes Ingress + IngressRoute so you can get pods
# healthy first; apply ingress when Traefik, cert-manager, and CRDs are ready.

set -euo pipefail

REAPER_NS="${REAPER_NS:-reaperc2-ns}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KUBECTL="${KUBECTL:-kubectl}"

die() { echo "error: $*" >&2; exit 1; }
info() { echo "==> $*"; }

usage() {
  cat <<'EOF'
Usage: ./deploy-aws-k8s.sh <command> [args]

Commands:
  help              Show this help.
  check-local       Exit 0 only if required *.local.yaml, bundled YAML, and ECR image in deployment.yaml are valid (no kubectl).
  preflight         Check kubectl context, local files (warnings), Traefik IngressClass, optional CRDs.
  fetch-ca          Run fetch-docdb-ca-bundle.sh (required before apply-core). Does not require *.local.yaml.
  apply-secrets     Apply namespace + all three examples/*.local.yaml + ../operator-ai.local.yaml (if present). Validates required files first (see check-local).
  ecr-secret        Create/update reaperc2-myregistrykey (validates required files first).
  apply-core        kubectl apply -k .  (app + SA + service; no ingress). Validates required files first.
  apply-ingress     Apply ingress.yaml + ingressroute.yaml (validates required files first).
  teardown-ingress  Remove Ingress, IngressRoute, TLS Secret (secretName from ingress.yaml), and cert-manager Certificate(s). Use before switching issuer (e.g. staging → prod).
  rollout           kubectl rollout restart + status (validates required files first).
  job-docdb-user    Re-run docdb-init-user job (validates required files first).
  job-docdb-init    Re-run docdb-init job (validates required files first).
  status            Pods, deployment, services.
  all               Require local prereqs, then preflight, fetch-ca, apply-secrets, ecr-secret, apply-core, rollout (then prints next steps).
  teardown          Remove app from cluster (same as README Teardown): kustomize stack, ingress, jobs, sample secret apply, ollama leftovers. Does not delete the namespace or DocumentDB.

Environment:
  REAPER_NS              Namespace (default: reaperc2-ns)
  REAPER_ECR_ACCOUNT     AWS account id for ECR (optional if deployment.yaml image is a real *.dkr.ecr.<region>.amazonaws.com URI)
  AWS_REGION             ECR region for aws ecr get-login-password (optional if derivable from deployment image)
  SKIP_ECR_SECRET        Set to 1 to skip ecr-secret in "all"

Note: teardown does not remove Operator AI objects; use kubectl delete -f ../operator-ai.local.yaml if needed.
EOF
}

cmd_help() { usage; }

cmd_preflight() {
  command -v "$KUBECTL" >/dev/null || die "kubectl not found"
  "$KUBECTL" version --client >/dev/null
  info "kubectl context: $($KUBECTL config current-context 2>/dev/null || echo '(none)')"
  test -f "$ROOT/rds-combined-ca-bundle.pem" || echo "warn: missing $ROOT/rds-combined-ca-bundle.pem — run: ./deploy-aws-k8s.sh fetch-ca"

  local missing=0
  for f in \
    "$ROOT/examples/documentdb-secret.local.yaml" \
    "$ROOT/examples/documentdb-admin-secret.local.yaml" \
    "$ROOT/examples/admin-bootstrap-secret.local.yaml"; do
    if [[ ! -f "$f" ]]; then
      echo "warn: missing $f (copy from examples/*.yaml templates)"
      missing=1
    fi
  done

  if ! "$KUBECTL" get ingressclass traefik -o name >/dev/null 2>&1; then
    echo "warn: IngressClass 'traefik' not found — apply-core is fine; apply-ingress will fail until Traefik is installed."
  else
    info "IngressClass traefik: ok"
  fi

  if ! "$KUBECTL" get crd ingressroutes.traefik.io >/dev/null 2>&1; then
    echo "warn: CRD ingressroutes.traefik.io not found — apply-ingress (IngressRoute) will fail until Traefik CRDs exist."
  else
    info "CRD ingressroutes.traefik.io: ok"
  fi

  if ! "$KUBECTL" get crd certificates.cert-manager.io >/dev/null 2>&1; then
    echo "warn: cert-manager CRDs not found — Ingress may stay Pending for TLS until cert-manager is installed."
  else
    info "CRD certificates.cert-manager.io: ok"
  fi

  [[ "$missing" -eq 0 ]] || echo "warn: copy missing secrets before apply-secrets (see README)."
}

cmd_fetch_ca() {
  test -x "$ROOT/fetch-docdb-ca-bundle.sh" || die "chmod +x fetch-docdb-ca-bundle.sh"
  (cd "$ROOT" && ./fetch-docdb-ca-bundle.sh)
}

cmd_apply_secrets() {
  require_aws_prereqs
  "$KUBECTL" apply -f "$ROOT/namespace.yaml"
  for f in \
    "$ROOT/examples/documentdb-secret.local.yaml" \
    "$ROOT/examples/documentdb-admin-secret.local.yaml" \
    "$ROOT/examples/admin-bootstrap-secret.local.yaml"; do
    [[ -f "$f" ]] || die "missing $f"
    "$KUBECTL" apply -f "$f"
  done
  if [[ -f "$ROOT/../operator-ai.local.yaml" ]]; then
    info "Applying ../operator-ai.local.yaml"
    "$KUBECTL" apply -f "$ROOT/../operator-ai.local.yaml"
  else
    echo "warn: ../operator-ai.local.yaml not found — Operator AI env not updated (copy from ../operator-ai.yaml)."
  fi
}

ecr_account_from_deployment() {
  local img
  img="$(grep -E '^\s+image:\s' "$ROOT/deployment.yaml" | head -1 | awk '{print $2}')"
  # 123456789012.dkr.ecr.us-east-1.amazonaws.com/reaperc2:tag
  if [[ "$img" =~ ^([0-9]{12})\.dkr\.ecr\.([a-z0-9-]+)\.amazonaws\.com/ ]]; then
    echo "${BASH_REMATCH[1]}"
    return
  fi
  return 1
}

ecr_region_from_deployment() {
  local img
  img="$(grep -E '^\s+image:\s' "$ROOT/deployment.yaml" | head -1 | awk '{print $2}')"
  if [[ "$img" =~ \.dkr\.ecr\.([a-z0-9-]+)\.amazonaws\.com ]]; then
    echo "${BASH_REMATCH[1]}"
    return
  fi
  return 1
}

# All cluster deploy paths call this first (not: fetch-ca, preflight, status, teardown, help).
require_aws_prereqs() {
  local missing=()
  local f
  for f in \
    "$ROOT/examples/documentdb-secret.local.yaml" \
    "$ROOT/examples/documentdb-admin-secret.local.yaml" \
    "$ROOT/examples/admin-bootstrap-secret.local.yaml" \
    "$ROOT/namespace.yaml" \
    "$ROOT/serviceaccount.yaml" \
    "$ROOT/deployment.yaml" \
    "$ROOT/service.yaml" \
    "$ROOT/kustomization.yaml" \
    "$ROOT/ingress.yaml" \
    "$ROOT/ingressroute.yaml" \
    "$ROOT/docdb-init-job.yaml" \
    "$ROOT/docdb-init-user-job.yaml"; do
    [[ -f "$f" ]] || missing+=("$f")
  done
  [[ -f "$ROOT/fetch-docdb-ca-bundle.sh" ]] || missing+=("$ROOT/fetch-docdb-ca-bundle.sh")
  if [[ ${#missing[@]} -gt 0 ]]; then
    echo "error: missing required files for AWS deploy:" >&2
    printf '  - %s\n' "${missing[@]}" >&2
    cat >&2 <<'EOM'
Create secrets from templates (see deployments/k8s/AWS/README.md):
  cp examples/documentdb-secret.yaml examples/documentdb-secret.local.yaml
  cp examples/documentdb-admin-secret.yaml examples/documentdb-admin-secret.local.yaml
  cp examples/admin-bootstrap-secret.yaml examples/admin-bootstrap-secret.local.yaml
Then edit hosts, passwords, and admin bootstrap credentials.
EOM
    exit 1
  fi
  [[ -x "$ROOT/fetch-docdb-ca-bundle.sh" ]] || die "run: chmod +x $ROOT/fetch-docdb-ca-bundle.sh"
  local acct
  acct="$(ecr_account_from_deployment || true)"
  [[ -n "$acct" && "$acct" != "123456789012" ]] || die "deployment.yaml image must be a real ECR URI (replace placeholder account 123456789012 with your AWS account id)."
}

cmd_check_local() {
  require_aws_prereqs
  info "Local prerequisites OK for AWS deploy."
}

cmd_ecr_secret() {
  require_aws_prereqs
  local acct region
  acct="${REAPER_ECR_ACCOUNT:-}"
  region="${AWS_REGION:-}"
  if [[ -z "$acct" ]]; then
    acct="$(ecr_account_from_deployment || true)"
  fi
  [[ -n "$acct" && "$acct" != "123456789012" ]] || die "Set REAPER_ECR_ACCOUNT or fix deployment.yaml image to your real ECR URI (not placeholder 123456789012)."
  if [[ -z "$region" ]]; then
    region="$(ecr_region_from_deployment || echo us-east-1)"
  fi
  command -v aws >/dev/null || die "aws CLI required for ecr-secret"
  info "ECR docker-registry secret for ${acct}.dkr.ecr.${region}.amazonaws.com"
  "$KUBECTL" create secret docker-registry reaperc2-myregistrykey \
    --namespace="$REAPER_NS" \
    --docker-server="${acct}.dkr.ecr.${region}.amazonaws.com" \
    --docker-username=AWS \
    --docker-password="$(aws ecr get-login-password --region "$region")" \
    --dry-run=client -o yaml | "$KUBECTL" apply -f -
}

cmd_apply_core() {
  require_aws_prereqs
  test -f "$ROOT/rds-combined-ca-bundle.pem" || die "run ./deploy-aws-k8s.sh fetch-ca first"
  (cd "$ROOT" && "$KUBECTL" apply -k .)
}

cmd_apply_ingress() {
  require_aws_prereqs
  cmd_preflight
  "$KUBECTL" apply -f "$ROOT/ingress.yaml" -f "$ROOT/ingressroute.yaml" -n "$REAPER_NS"
  info "Ingress applied. Check: kubectl describe ingress -n $REAPER_NS reaperc2-ingress"
}

cmd_rollout() {
  require_aws_prereqs
  "$KUBECTL" rollout restart deployment/reaperc2-deployment -n "$REAPER_NS"
  "$KUBECTL" rollout status deployment/reaperc2-deployment -n "$REAPER_NS" --timeout=300s
}

cmd_job_docdb_user() {
  require_aws_prereqs
  "$KUBECTL" delete job docdb-init-user -n "$REAPER_NS" --ignore-not-found
  "$KUBECTL" apply -f "$ROOT/docdb-init-user-job.yaml"
  "$KUBECTL" wait -n "$REAPER_NS" job/docdb-init-user --for=condition=complete --timeout=180s
  "$KUBECTL" logs -n "$REAPER_NS" job/docdb-init-user --tail=80
}

cmd_job_docdb_init() {
  require_aws_prereqs
  "$KUBECTL" delete job docdb-init -n "$REAPER_NS" --ignore-not-found
  "$KUBECTL" apply -f "$ROOT/docdb-init-job.yaml"
  "$KUBECTL" wait -n "$REAPER_NS" job/docdb-init --for=condition=complete --timeout=180s
  "$KUBECTL" logs -n "$REAPER_NS" job/docdb-init --tail=80
}

cmd_status() {
  "$KUBECTL" get pods,svc,deploy -n "$REAPER_NS"
  "$KUBECTL" get ingress,ingressroute -n "$REAPER_NS" 2>/dev/null || true
}

cmd_teardown_ingress() {
  command -v "$KUBECTL" >/dev/null || die "kubectl not found"
  [[ -f "$ROOT/ingress.yaml" ]] || die "missing $ROOT/ingress.yaml"
  [[ -f "$ROOT/ingressroute.yaml" ]] || die "missing $ROOT/ingressroute.yaml"

  local tls_secret ing_name
  tls_secret="$(grep -E '^[[:space:]]*secretName:' "$ROOT/ingress.yaml" | head -1 | awk '{print $2}')"
  [[ -n "${tls_secret:-}" ]] || die "could not parse tls.secretName from ingress.yaml"
  ing_name="$(grep -E '^[[:space:]]*name:' "$ROOT/ingress.yaml" | head -1 | awk '{print $2}')"
  [[ -n "${ing_name:-}" ]] || ing_name=reaperc2-ingress

  info "Deleting Ingress and IngressRoute in namespace $REAPER_NS"
  "$KUBECTL" delete -f "$ROOT/ingress.yaml" -f "$ROOT/ingressroute.yaml" -n "$REAPER_NS" --ignore-not-found

  if "$KUBECTL" get crd certificates.cert-manager.io >/dev/null 2>&1; then
    info "Deleting cert-manager Certificate(s) for ingress $ing_name (if any)"
    "$KUBECTL" delete certificate -n "$REAPER_NS" -l "cert-manager.io/ingress-name=${ing_name}" --ignore-not-found
    "$KUBECTL" delete certificate -n "$REAPER_NS" "$tls_secret" --ignore-not-found
  else
    echo "warn: CRD certificates.cert-manager.io not found — skip Certificate delete"
  fi

  info "Deleting TLS Secret $tls_secret"
  "$KUBECTL" delete secret "$tls_secret" -n "$REAPER_NS" --ignore-not-found

  cat <<EOF
Ingress TLS stack removed. Edit ingress.yaml (e.g. cert-manager.io/cluster-issuer: letsencrypt-prod), then:
  ./deploy-aws-k8s.sh apply-ingress
EOF
}

cmd_teardown() {
  info "Teardown: kustomize stack in $ROOT"
  (cd "$ROOT" && "$KUBECTL" delete -k . --ignore-not-found)
  "$KUBECTL" delete -f "$ROOT/ingress.yaml" -f "$ROOT/ingressroute.yaml" -n "$REAPER_NS" --ignore-not-found
  if [[ -f "$ROOT/examples/documentdb-secret.local.yaml" ]]; then
    "$KUBECTL" delete -f "$ROOT/examples/documentdb-secret.local.yaml" --ignore-not-found
  else
    echo "warn: $ROOT/examples/documentdb-secret.local.yaml missing — skip deleting that Secret (delete manually if applied)"
  fi
  "$KUBECTL" delete -f "$ROOT/docdb-init-job.yaml" -f "$ROOT/docdb-init-user-job.yaml" --ignore-not-found
  "$KUBECTL" delete deployment/ollama service/ollama pvc/ollama-data -n "$REAPER_NS" --ignore-not-found
  info "Done. Namespace $REAPER_NS and other Secrets (admin bootstrap, ECR pull, Operator AI) may still exist — delete manually if needed."
}

cmd_all() {
  require_aws_prereqs
  cmd_preflight
  cmd_fetch_ca
  cmd_apply_secrets
  if [[ "${SKIP_ECR_SECRET:-0}" != "1" ]]; then
    cmd_ecr_secret
  else
    info "Skipping ecr-secret (SKIP_ECR_SECRET=1)"
  fi
  cmd_apply_core
  cmd_rollout
  cat <<EOF

Next steps (see README):
  1) DocumentDB app user password:  ./deploy-aws-k8s.sh job-docdb-user
  2) Collections / indexes (optional):  ./deploy-aws-k8s.sh job-docdb-init
  3) When Traefik + cert-manager + IngressRoute CRD are ready:  ./deploy-aws-k8s.sh apply-ingress
  4) Admin UI:  kubectl port-forward -n $REAPER_NS deployment/reaperc2-deployment 8443:8443
EOF
}

main() {
  local cmd="${1:-help}"
  shift || true
  case "$cmd" in
    help|-h|--help) cmd_help ;;
    check-local) cmd_check_local ;;
    preflight) cmd_preflight ;;
    fetch-ca) cmd_fetch_ca ;;
    apply-secrets) cmd_apply_secrets ;;
    ecr-secret) cmd_ecr_secret ;;
    apply-core) cmd_apply_core ;;
    apply-ingress) cmd_apply_ingress ;;
    rollout) cmd_rollout ;;
    job-docdb-user) cmd_job_docdb_user ;;
    job-docdb-init) cmd_job_docdb_init ;;
    status) cmd_status ;;
    all) cmd_all ;;
    teardown) cmd_teardown ;;
    teardown-ingress) cmd_teardown_ingress ;;
    *) die "unknown command: $cmd (try: ./deploy-aws-k8s.sh help)" ;;
  esac
}

main "$@"
