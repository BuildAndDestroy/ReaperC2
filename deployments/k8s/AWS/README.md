# AWS: Deploy ReaperC2 on EKS

Kubernetes manifests for ReaperC2 on an **existing** AWS stack:

- **EKS** cluster (kubeconfig already configured)
- **ECR** image for ReaperC2
- **Amazon DocumentDB** (endpoint and credentials from your infra repo)
- **Traefik** ingress controller and load balancer

Infrastructure provisioning (VPC, EKS, DocumentDB, Traefik install) lives in a separate repository. This directory only deploys the application.

## Prerequisites

- `kubectl` pointed at your cluster (`aws eks update-kubeconfig ...` if needed)
- ReaperC2 image built and pushed to ECR (see [Build and push](#build-and-push) below)
- DocumentDB cluster endpoint and application DB user (`api_user` / `api_db` or your naming)
- Traefik installed with an `IngressClass` named `traefik` (adjust manifests if yours differs)

## Build and push

From the repo root, build **linux/amd64** and **linux/arm64** and push a multi-arch manifest to ECR (defaults match `deployment.yaml`; override with env vars):

```bash
export AWS_ACCESS_KEY_ID=AKIA...
export AWS_SECRET_ACCESS_KEY=...
make build
# Or: make build AWS_CLI_PROFILE=your-sso-profile
# Or pin a release tag:
make build IMAGE_TAG=v1.0.0
```

Requires Docker with **buildx**, the **AWS CLI**, and ECR permissions. The Makefile runs `git submodule update --init --recursive` before build so Scythe matches this repo.

| Target | Purpose |
|--------|---------|
| `make build` | Multi-arch (`amd64` + `arm64`) push to `$(ECR_REGISTRY)/reaperc2:$(IMAGE_TAG)` |
| `make build-amd64` | Push `...:$(IMAGE_TAG)-amd64` only |
| `make build-arm64` | Push `...:$(IMAGE_TAG)-arm64` only |
| `make build-local` | Load single-arch image `reaperc2:local` (no ECR push) |

Variables: `AWS_ACCOUNT_ID`, `AWS_REGION`, `ECR_REPOSITORY`, `IMAGE_TAG`, `SCYTHE_GIT_REF`. Run `make help` for defaults.

Then set `deployment.yaml` `image:` to the tag you pushed (e.g. `123456789012.dkr.ecr.us-east-1.amazonaws.com/reaperc2:abc1234` — replace with your AWS account ID).

## Configure before apply

| Item | File |
|------|------|
| ECR image URI | `deployment.yaml` → `spec.template.spec.containers[0].image` |
| Beacon hostname / TLS | `ingress.yaml`, `ingressroute.yaml` → `subdomain.domain.com` |
| DocumentDB host, user, password | `examples/documentdb-secret.yaml` |
| ECR pull secret | `examples/registry-secret.yaml` (commands only) |

## Deploy

### 1. DocumentDB TLS CA bundle

ReaperC2 uses `DEPLOY_ENV=AWS`, which enables DocumentDB TLS in [`pkg/dbconnections/mongoconnections.go`](../../../pkg/dbconnections/mongoconnections.go). Download the RDS trust bundle:

```bash
cd deployments/k8s/AWS
chmod +x fetch-docdb-ca-bundle.sh
./fetch-docdb-ca-bundle.sh
```

### 2. Secrets

Edit and apply DocumentDB credentials (from your infra repo / secrets manager):

```bash
kubectl apply -f examples/documentdb-secret.yaml
```

Create the ECR pull secret (see `examples/registry-secret.yaml`), for example:

```bash
kubectl create secret docker-registry reaperc2-myregistrykey \
  --namespace=reaperc2-ns \
  --docker-server=ACCOUNT.dkr.ecr.us-east-1.amazonaws.com \
  --docker-username=AWS \
  --docker-password="$(aws ecr get-login-password --region us-east-1)" \
  --dry-run=client -o yaml | kubectl apply -f -
```

### 3. ReaperC2 + Ollama

```bash
kubectl apply -k .
```

This applies ReaperC2, **in-cluster Ollama** (`gpt-oss:latest` on a 50 Gi PVC), and `reaperc2-ai-config` pointing Operator AI at `http://ollama:11434/v1`. The Ollama init container pulls the model on first install (can take several minutes); watch with:

```bash
kubectl logs -n reaperc2-ns deployment/ollama -c pull-model -f
kubectl wait -n reaperc2-ns deployment/ollama --for=condition=Available --timeout=600s
```

Tune CPU/memory and PVC size in `ollama.yaml` if `gpt-oss` needs more than 16 Gi RAM.

**Cloud API keys (optional):** create Secret `reaperc2-ai-secrets` with OpenAI/Anthropic/Bedrock keys. Do **not** apply the root [`operator-ai.yaml`](../operator-ai.yaml) ConfigMap if you use this bundle’s `ai-config.yaml` (it would overwrite Ollama settings). Example:

```bash
kubectl create secret generic reaperc2-ai-secrets -n reaperc2-ns \
  --from-literal=REAPER_AI_OPENAI_API_KEY=sk-... \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl rollout restart deployment/reaperc2-deployment -n reaperc2-ns
```

### 4. Verify

```bash
kubectl get pods,svc,ingress,ingressroute -n reaperc2-ns
kubectl logs -n reaperc2-ns deployment/reaperc2-deployment --tail=50
kubectl exec -n reaperc2-ns deployment/ollama -- ollama list
```

Admin UI (not on public ingress):

```bash
kubectl port-forward -n reaperc2-ns deployment/reaperc2-deployment 8443:8443
```

Open `http://127.0.0.1:8443` locally.

## Optional: DocumentDB debug pod

For `mongosh` or one-off DB work from inside the cluster:

```bash
kubectl apply -f docdb-mongosh.yaml
kubectl exec -it -n reaperc2-ns docdb-tester -- bash
# CA bundle is mounted at /certs/rds-combined-ca-bundle.pem
```

To seed collections, adapt [`test/setup_mongo.sh`](../../../test/setup_mongo.sh) with your DocumentDB endpoint and TLS, or run equivalent `mongosh` commands from a host or pod that can reach DocumentDB.

## File layout

| File | Purpose |
|------|---------|
| `kustomization.yaml` | `kubectl apply -k` entrypoint |
| `namespace.yaml` | `reaperc2-ns` |
| `ollama.yaml` | Ollama Deployment, Service, PVC (`gpt-oss:latest`) |
| `ai-config.yaml` | Operator AI ConfigMap (Ollama in-cluster) |
| `deployment.yaml` | ReaperC2 + DocumentDB env + CA volume |
| `service.yaml` | ClusterIP :8080 (beacon) |
| `ingress.yaml` | Standard Ingress for Traefik / cert-manager |
| `ingressroute.yaml` | Traefik `IngressRoute` (beacon :8080) |
| `fetch-docdb-ca-bundle.sh` | Downloads RDS global CA PEM |
| `docdb-mongosh.yaml` | Optional debug pod |
| `examples/documentdb-secret.yaml` | DocumentDB connection secret |
| `examples/registry-secret.yaml` | ECR pull secret instructions |

## Teardown (app only)

```bash
kubectl delete -k . --ignore-not-found
kubectl delete -f examples/documentdb-secret.yaml --ignore-not-found
kubectl delete -f docdb-mongosh.yaml --ignore-not-found
```
