#!/usr/bin/env bash
# Shared helpers for deployments/k3s/scripts (OnPrem NFS + Ollama).

set -euo pipefail

kubectl_cmd() {
  if command -v kubectl >/dev/null 2>&1; then
    kubectl "$@"
  elif command -v k3s >/dev/null 2>&1; then
    sudo k3s kubectl "$@"
  else
    echo "kubectl or k3s not found" >&2
    exit 1
  fi
}

require_file() {
  local f="$1"
  local hint="$2"
  if [[ -f "$f" ]]; then
    return 0
  fi
  echo "Missing required file: $f" >&2
  echo "$hint" >&2
  exit 1
}

# Load the first existing config.env from candidate paths.
# Sets DEPLOY_CONFIG_FILE to the file sourced (empty if none found).
source_deploy_config() {
  DEPLOY_CONFIG_FILE=""
  DEPLOY_CONFIG_CANDIDATES=("$@")
  local f
  for f in "${DEPLOY_CONFIG_CANDIDATES[@]}"; do
    if [[ -f "$f" ]]; then
      # shellcheck disable=SC1090
      set -a
      source "$f"
      set +a
      DEPLOY_CONFIG_FILE="$f"
      if [[ "$f" != "${DEPLOY_CONFIG_CANDIDATES[0]}" ]]; then
        echo "Note: using ${f} (create ${DEPLOY_CONFIG_CANDIDATES[0]} to override)" >&2
      fi
      return 0
    fi
  done
  return 1
}

require_registry_credentials() {
  if [[ -n "${REGISTRY_USERNAME:-}" && -n "${REGISTRY_PASSWORD:-}" ]]; then
    return 0
  fi
  echo "Set REGISTRY_USERNAME and REGISTRY_PASSWORD in config.env." >&2
  if [[ -n "${DEPLOY_CONFIG_FILE:-}" ]]; then
    echo "Loaded ${DEPLOY_CONFIG_FILE} but registry credentials are missing or empty." >&2
    echo "Quote values that contain # or ! — e.g. REGISTRY_PASSWORD='your-pass'" >&2
  else
    echo "No config.env found. Create one of:" >&2
    local f
    for f in "${DEPLOY_CONFIG_CANDIDATES[@]}"; do
      echo "  $f" >&2
    done
    echo "Copy from config.example.env in the same directory." >&2
  fi
  exit 1
}

apply_ollama_pull_models() {
  if [[ -z "${OLLAMA_PULL_MODELS:-}" ]]; then
    return 0
  fi
  kubectl_cmd create configmap ollama-pull-models \
    --from-literal="OLLAMA_PULL_MODELS=${OLLAMA_PULL_MODELS}" \
    -n ollama-ns \
    --dry-run=client -o yaml | kubectl_cmd apply -f -
}

patch_mongo_nfs_pv() {
  local pv_file="$1"
  local server="$2"
  local path="$3"
  if [[ ! -f "$pv_file" ]]; then
    echo "PV file not found: $pv_file" >&2
    exit 1
  fi
  sed -e "s|server: .*|server: ${server}|" \
      -e "s|path: /export/.*|path: ${path}|" \
      "$pv_file" | kubectl_cmd apply -f -
}

apply_ollama_stack() {
  local k8s_root="$1"
  local k3s_root="${2:-}"
  local sc="${OLLAMA_STORAGE_CLASS:-local-path}"
  echo "Applying Ollama (Operator AI backend)..."
  kubectl_cmd create namespace ollama-ns --dry-run=client -o yaml | kubectl_cmd apply -f -
  if [[ -n "${OLLAMA_NFS_SERVER:-}" && -n "${OLLAMA_NFS_PATH:-}" && -n "$k3s_root" ]]; then
    sed -e "s|server: .*|server: ${OLLAMA_NFS_SERVER}|" \
        -e "s|path: /export/.*|path: ${OLLAMA_NFS_PATH}|" \
        "${k3s_root}/ollama-nfs-pv.yaml" | kubectl_cmd apply -f -
    kubectl_cmd apply -f "${k3s_root}/ollama-pvc-nfs.yaml"
    sed -e '/^kind: PersistentVolumeClaim$/,/^---$/d' \
        "${k8s_root}/ollama.yaml" | kubectl_cmd apply -f -
  elif [[ -n "$sc" ]]; then
    sed -e "s|storageClassName: gp2|storageClassName: ${sc}|" \
        "${k8s_root}/ollama.yaml" | kubectl_cmd apply -f -
  else
    sed -e '/^[[:space:]]*storageClassName: gp2/d' \
        "${k8s_root}/ollama.yaml" | kubectl_cmd apply -f -
  fi
  apply_ollama_pull_models
  apply_ollama_api_key_secret
  echo "Waiting for Ollama PVC..."
  kubectl_cmd -n ollama-ns wait --for=jsonpath='{.status.phase}'=Bound pvc/ollama-data --timeout=180s || true
  kubectl_cmd -n ollama-ns rollout status deploy/ollama --timeout=3600s || true
}

apply_operator_ai() {
  local k8s_root="$1"
  local ns="${REAPERC2_NAMESPACE:-reaperc2-ns}"
  echo "Applying Operator AI config..."
  kubectl_cmd apply -f "${k8s_root}/operator-ai.yaml"
  local ai_patch='{"data":{"REAPER_AI_OLLAMA_ENABLED":"0","REAPER_AI_DEFAULT_PROVIDER":"bedrock"}}'
  if kubectl_cmd get deploy ollama -n ollama-ns >/dev/null 2>&1; then
    ai_patch='{"data":{"REAPER_AI_OLLAMA_ENABLED":"1","REAPER_AI_OLLAMA_DISCOVER":"1","REAPER_AI_DEFAULT_PROVIDER":"ollama"}}'
    echo "In-cluster Ollama detected — Operator AI will use Ollama provider."
  fi
  kubectl_cmd patch configmap reaperc2-ai-config -n "$ns" \
    --type merge -p "$ai_patch" 2>/dev/null || true
}

apply_ollama_api_key_secret() {
  if [[ -z "${OLLAMA_API_KEY:-}" ]]; then
    return 0
  fi
  kubectl_cmd create secret generic ollama-api-key \
    --from-literal=OLLAMA_API_KEY="${OLLAMA_API_KEY}" \
    -n ollama-ns \
    --dry-run=client -o yaml | kubectl_cmd apply -f -
}

prepare_reaperc2_build() {
  local repo_root="$1"
  local arch="$2"

  if ! command -v go >/dev/null 2>&1; then
    echo "Go is required on the host to cross-compile ReaperC2. Install Go or use: make build-binaries" >&2
    exit 1
  fi

  echo "Preparing source (submodule + vendor + linux/${arch} binary)..."
  (
    cd "$repo_root"
    git submodule update --init --recursive
    if [[ ! -d vendor ]] || [[ go.mod -nt vendor ]]; then
      echo "Running go mod vendor..."
      go mod vendor
    fi
    mkdir -p "bin/linux-${arch}"
    CGO_ENABLED=0 GOOS=linux GOARCH="${arch}" \
      go build -mod=vendor -trimpath -ldflags="-s -w" \
      -o "bin/linux-${arch}/ReaperC2" ./cmd
  )
}

docker_build_push() {
  local repo_root="$1"
  local image="$2"
  local platform="$3"
  local import_local="${4:-false}"
  local arch="${platform#linux/}"
  local scythe_ref="${SCYTHE_GIT_REF:-main}"
  local load_flag=()
  local push_flag=(--push)

  if [[ "${DOCKER_BUILD_LOAD:-}" == "1" ]]; then
    load_flag=(--load)
    push_flag=()
  fi

  prepare_reaperc2_build "$repo_root" "$arch"

  echo "Building ${image} for ${platform}..."
  docker buildx build \
    --platform "${platform}" \
    -t "${image}" \
    -f "${repo_root}/Dockerfile.pack" \
    --build-arg "TARGETARCH=${arch}" \
    --build-arg "SCYTHE_GIT_REF=${scythe_ref}" \
    "${push_flag[@]}" \
    "${load_flag[@]}" \
    "${repo_root}"

  if [[ "$import_local" == true ]]; then
    local tar
    tar="$(mktemp /tmp/reaperc2-XXXXXX.tar)"
    trap 'rm -f "$tar"' RETURN
    docker pull --platform "${platform}" "${image}"
    docker save "${image}" -o "$tar"
    if command -v k3s >/dev/null 2>&1; then
      sudo k3s ctr images import "$tar"
    else
      echo "k3s not found; image pushed to registry only." >&2
    fi
  fi
}

create_registry_pull_secret() {
  local namespace="$1"
  local secret_name="$2"
  local namespace_file="$3"

  kubectl_cmd apply -f "$namespace_file"
  kubectl_cmd create secret docker-registry "$secret_name" \
    --docker-server="${REGISTRY_SERVER}" \
    --docker-username="${REGISTRY_USERNAME}" \
    --docker-password="${REGISTRY_PASSWORD}" \
    -n "$namespace" \
    --dry-run=client -o yaml | kubectl_cmd apply -f -
}

rollout_reaperc2() {
  local namespace="$1"
  kubectl_cmd -n "$namespace" rollout restart deployment/reaperc2-deployment
  kubectl_cmd -n "$namespace" rollout status deployment/reaperc2-deployment --timeout=300s
}

patch_k3s_ingress_host() {
  local file="$1"
  local host="$2"
  local issuer="$3"
  sed -e "s|metrics\\.harvestrangelabs\\.com|${host}|g" \
      -e "s|cert-manager.io/cluster-issuer: .*|cert-manager.io/cluster-issuer: ${issuer}|" \
      "$file"
}

apply_k3s_ingress() {
  local k3s_root="$1"
  local host="${INGRESS_HOST:-metrics.harvestrangelabs.com}"
  local issuer="${CERT_MANAGER_ISSUER:-letsencrypt-prod}"
  local ns="${REAPERC2_NAMESPACE:-reaperc2-ns}"

  if ! kubectl_cmd get clusterissuer "$issuer" >/dev/null 2>&1; then
    echo "ClusterIssuer ${issuer} not found — install cert-manager and create the issuer first." >&2
    exit 1
  fi

  echo "Applying beacon ingress for ${host} (admin UI remains port-forward only)..."
  patch_k3s_ingress_host "${k3s_root}/traefik-redirect-https-middleware.yaml" "$host" "$issuer" \
    | kubectl_cmd apply -f -
  patch_k3s_ingress_host "${k3s_root}/traefik-security-middleware.yaml" "$host" "$issuer" \
    | kubectl_cmd apply -f -
  patch_k3s_ingress_host "${k3s_root}/ingress-http-redirect.yaml" "$host" "$issuer" \
    | kubectl_cmd apply -f -
  patch_k3s_ingress_host "${k3s_root}/ingress.yaml" "$host" "$issuer" \
    | kubectl_cmd apply -f -
  patch_k3s_ingress_host "${k3s_root}/ingressroute.yaml" "$host" "$issuer" \
    | kubectl_cmd apply -f -
  kubectl_cmd -n "$ns" get ingress,ingressroute 2>/dev/null || true
}

teardown_k3s_ingress() {
  local k3s_root="$1"
  local ns="${REAPERC2_NAMESPACE:-reaperc2-ns}"
  local tls_secret="metrics-harvestrangelabs-com-tls"
  local ing_name="reaperc2-ingress"

  echo "Removing ingress, middlewares, and staging/prod TLS resources..."
  kubectl_cmd delete -f "${k3s_root}/ingress.yaml" -f "${k3s_root}/ingressroute.yaml" \
    -f "${k3s_root}/ingress-http-redirect.yaml" \
    -f "${k3s_root}/traefik-redirect-https-middleware.yaml" \
    -f "${k3s_root}/traefik-security-middleware.yaml" \
    -n "$ns" --ignore-not-found

  if kubectl_cmd get crd certificates.cert-manager.io >/dev/null 2>&1; then
    kubectl_cmd delete certificate -n "$ns" -l "cert-manager.io/ingress-name=${ing_name}" --ignore-not-found
    kubectl_cmd delete certificate -n "$ns" "$tls_secret" --ignore-not-found
    kubectl_cmd delete certificaterequest -n "$ns" --all --ignore-not-found
  fi
  kubectl_cmd delete secret "$tls_secret" -n "$ns" --ignore-not-found
  kubectl_cmd delete pod -n "$ns" -l acme.cert-manager.io/http01-solver=true --ignore-not-found
}
