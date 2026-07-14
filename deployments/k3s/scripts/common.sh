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
  sed -e "s|server: 192.168.1.100|server: ${server}|" \
      -e "s|path: /volume1/reaperc2db|path: ${path}|" \
      "$pv_file" | kubectl_cmd apply -f -
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
    docker pull "${image}"
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

apply_ollama_stack() {
  local k8s_root="$1"
  echo "Applying Ollama (Operator AI backend)..."
  kubectl_cmd apply -f "${k8s_root}/ollama.yaml"
  apply_ollama_pull_models
  echo "Waiting for Ollama PVC..."
  kubectl_cmd -n ollama-ns wait --for=jsonpath='{.status.phase}'=Bound pvc/ollama-data --timeout=180s || true
  kubectl_cmd -n ollama-ns rollout status deploy/ollama --timeout=900s || true
}

apply_operator_ai() {
  local k8s_root="$1"
  echo "Applying Operator AI config..."
  kubectl_cmd apply -f "${k8s_root}/operator-ai.yaml"
}

rollout_reaperc2() {
  local namespace="$1"
  kubectl_cmd -n "$namespace" rollout restart deployment/reaperc2-deployment
  kubectl_cmd -n "$namespace" rollout status deployment/reaperc2-deployment --timeout=300s
}
