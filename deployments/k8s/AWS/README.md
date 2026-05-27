# AWS: Deploy ReaperC2 on EKS

Kubernetes manifests for ReaperC2 on an **existing** AWS stack:

- **EKS** cluster (kubeconfig already configured)
- **ECR** image for ReaperC2
- **Amazon DocumentDB** (endpoint and credentials from your infra repo)
- **Traefik** ingress controller and load balancer

Infrastructure provisioning (VPC, EKS, DocumentDB, Traefik install) lives in a separate repository. This directory only deploys the application.

## Run from scratch (checklist)

From the **repo root**, after `make build` and editing manifests (image URI, ingress hostnames, DocumentDB secret):

```bash
cd deployments/k8s/AWS

# 1. RDS/DocumentDB TLS CA (required once per machine; file is gitignored)
chmod +x fetch-docdb-ca-bundle.sh
./fetch-docdb-ca-bundle.sh

# 2. DocumentDB secret (templates in examples/ — use *.local.yaml for real values; see examples/README.md)
cp examples/documentdb-secret.yaml examples/documentdb-secret.local.yaml
# edit examples/documentdb-secret.local.yaml
kubectl apply -f namespace.yaml
kubectl apply -f examples/documentdb-secret.local.yaml

# 3. ECR pull secret (replace ACCOUNT and region)
kubectl create secret docker-registry reaperc2-myregistrykey \
  --namespace=reaperc2-ns \
  --docker-server=ACCOUNT.dkr.ecr.us-east-1.amazonaws.com \
  --docker-username=AWS \
  --docker-password="$(aws ecr get-login-password --region us-east-1)" \
  --dry-run=client -o yaml | kubectl apply -f -

# 4. Optional: create app DB user if it does not exist in DocumentDB yet
#    cp examples/documentdb-admin-secret.yaml examples/documentdb-admin-secret.local.yaml && edit, then:
kubectl apply -f examples/documentdb-admin-secret.local.yaml
kubectl apply -k .    # namespace, ConfigMaps (CA + init scripts), ReaperC2, ingress
kubectl apply -f docdb-init-user-job.yaml
kubectl wait -n reaperc2-ns job/docdb-init-user --for=condition=complete --timeout=120s
kubectl logs -n reaperc2-ns job/docdb-init-user

# 5. DocumentDB collections + indexes (idempotent; no sample beacon data)
kubectl apply -f docdb-init-job.yaml
kubectl wait -n reaperc2-ns job/docdb-init --for=condition=complete --timeout=120s
kubectl logs -n reaperc2-ns job/docdb-init

# 6. If you skipped step 4, apply the app stack now:
# kubectl apply -k .

# 7. Verify
kubectl get pods,svc,ingress -n reaperc2-ns
kubectl logs -n reaperc2-ns deployment/reaperc2-deployment --tail=50
kubectl port-forward -n reaperc2-ns deployment/reaperc2-deployment 8443:8443
```

**Re-run a failed init Job:** delete the Job, then apply again:

```bash
kubectl delete job docdb-init -n reaperc2-ns --ignore-not-found
kubectl apply -f docdb-init-job.yaml
```

**Auth against `admin`:** if the app user authenticates with `authSource=admin`, uncomment `MONGO_AUTH_SOURCE` in `docdb-init-job.yaml` (and set the same on `deployment.yaml` if needed).

Steps 4–6 are expanded below. If the app user already exists in DocumentDB (created by Terraform/infra), skip step 4 and run step 5 after `kubectl apply -k .`.

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
| DocumentDB host, user, password | `examples/documentdb-secret.local.yaml` (from template; not committed) |
| ECR pull secret | `examples/registry-secret.yaml` (commands only) |
| Operator AI (Bedrock, etc.) | `ai-config.yaml` + Secret `reaperc2-ai-secrets` |

## Deploy (step by step)

### 1. `fetch-docdb-ca-bundle.sh`

ReaperC2 uses `DEPLOY_ENV=AWS`, which enables DocumentDB TLS in [`pkg/dbconnections/mongoconnections.go`](../../../pkg/dbconnections/mongoconnections.go). The script downloads the RDS global PEM into `rds-combined-ca-bundle.pem` (gitignored). Kustomize bakes it into ConfigMap `docdb-ca-cert` on `kubectl apply -k .`.

```bash
cd deployments/k8s/AWS
chmod +x fetch-docdb-ca-bundle.sh
./fetch-docdb-ca-bundle.sh
```

Re-run this when AWS rotates CA bundles or on a fresh clone before the first `kubectl apply -k .`.

### 2. Secrets

Copy [`examples/documentdb-secret.yaml`](examples/documentdb-secret.yaml) to `examples/documentdb-secret.local.yaml` and edit (see [`examples/README.md`](examples/README.md)). Keys are **`host`**, **`port`**, **`username`**, **`password`**, **`database`**, optional **`auth_source`** — ReaperC2 builds the Mongo URI in Go; do not put a full `mongodb-uri` in the secret.

```bash
kubectl apply -f namespace.yaml
kubectl apply -f examples/documentdb-secret.local.yaml
```

ECR pull secret: see `examples/registry-secret.yaml` or the [checklist](#run-from-scratch-checklist) above.

**Optional app user Job** (skip if infra already created `api_user` / your DB user):

```bash
kubectl apply -f examples/documentdb-admin-secret.local.yaml
kubectl apply -k .
kubectl apply -f docdb-init-user-job.yaml
kubectl wait -n reaperc2-ns job/docdb-init-user --for=condition=complete --timeout=120s
kubectl logs -n reaperc2-ns job/docdb-init-user
```

### 3. DocumentDB init Job (`docdb-init-job.yaml`)

Requires: CA bundle on disk, `reaperc2-documentdb-credentials` secret, and ConfigMaps from `kubectl apply -k .` (`docdb-ca-cert`, `docdb-init-scripts`). Idempotent — no sample beacon data.

```bash
kubectl apply -f docdb-init-job.yaml
kubectl wait -n reaperc2-ns job/docdb-init --for=condition=complete --timeout=120s
kubectl logs -n reaperc2-ns job/docdb-init
```

Collections: `clients`, `heartbeat`, `data`, `operators`, `operator_sessions`, `engagements`, `audit_logs`, `file_artifacts`, `operator_mfa_challenges`, `beacon_profiles`, `operator_chat`, with indexes aligned to [`pkg/dbconnections`](../../../pkg/dbconnections). ReaperC2 also creates admin/portal indexes on first start if you skip this Job.

### 4. ReaperC2 (`kubectl apply -k .`)

```bash
kubectl apply -k .
```

This applies ReaperC2, DocumentDB TLS ConfigMaps, ingress, and `reaperc2-ai-config` (Operator AI defaults to **AWS Bedrock** — no in-cluster Ollama).

**Operator AI:** edit `ai-config.yaml` (region, model IDs), then create `reaperc2-ai-secrets` with API keys or use Bedrock IAM/IRSA. Do **not** apply the root [`operator-ai.yaml`](../operator-ai.yaml) ConfigMap — it would overwrite this bundle’s `ai-config.yaml`.

```bash
kubectl create secret generic reaperc2-ai-secrets -n reaperc2-ns \
  --from-literal=REAPER_AI_BEDROCK_ACCESS_KEY_ID=... \
  --from-literal=REAPER_AI_BEDROCK_SECRET_ACCESS_KEY=... \
  --from-literal=REAPER_AI_OPENAI_API_KEY=sk-... \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl rollout restart deployment/reaperc2-deployment -n reaperc2-ns
```

See [Operator AI](../../../docs/operator-guide-ai.md) for Bedrock inference profile IDs and IAM.

If you previously deployed in-cluster Ollama, remove leftovers:

```bash
kubectl delete deployment/ollama service/ollama pvc/ollama-data -n reaperc2-ns --ignore-not-found
```

### 5. Verify

```bash
kubectl get pods,svc,ingress,ingressroute -n reaperc2-ns
kubectl logs -n reaperc2-ns deployment/reaperc2-deployment --tail=50
```

Admin UI (not on public ingress):

```bash
kubectl port-forward -n reaperc2-ns deployment/reaperc2-deployment 8443:8443
```

Open `http://127.0.0.1:8443` locally.

## File layout

| File | Purpose |
|------|---------|
| `kustomization.yaml` | `kubectl apply -k` entrypoint |
| `namespace.yaml` | `reaperc2-ns` |
| `ai-config.yaml` | Operator AI ConfigMap (Bedrock / cloud; no Ollama) |
| `deployment.yaml` | ReaperC2 + DocumentDB env + CA volume |
| `service.yaml` | ClusterIP :8080 (beacon) |
| `ingress.yaml` | Standard Ingress for Traefik / cert-manager |
| `ingressroute.yaml` | Traefik `IngressRoute` (beacon :8080) |
| `fetch-docdb-ca-bundle.sh` | Downloads RDS global CA PEM |
| `docdb-init-job.yaml` | One-shot Job: collections + indexes |
| `docdb-init-user-job.yaml` | Optional Job: create app DB user via master creds |
| `scripts/docdb-init.js` | Idempotent schema script (used by init Job) |
| `examples/documentdb-secret.yaml` | DocumentDB secret **template** (placeholders) |
| `examples/documentdb-secret.local.yaml` | Your real secret (gitignored) |
| `examples/documentdb-admin-secret.yaml` | Admin secret **template** for init-user Job |
| `examples/README.md` | How to copy templates → `.local.yaml` |
| `examples/registry-secret.yaml` | ECR pull secret instructions |

## Troubleshooting

### `reaperc2-deployment` CrashLoopBackOff

Confirm the error (almost always MongoDB on startup):

```bash
kubectl logs -n reaperc2-ns deployment/reaperc2-deployment --previous --tail=30
```

| Log line | Fix |
|----------|-----|
| `MongoDB Ping Error` / `authentication failed` | **Wrong `authSource`:** set `auth_source` in the secret to your DB name (e.g. `reaperc2_db`) or `kubectl set env deployment/reaperc2-deployment -n reaperc2-ns MONGO_AUTH_SOURCE=reaperc2_db`. **Password mismatch:** the DocumentDB user password must match the secret — fix the secret, then drop/recreate the user with `docdb-init-user-job.yaml` (see below) or update the password with your infra tooling. |
| `MONGO_HOST` / `MONGO_PASSWORD` required | Secret `reaperc2-documentdb-credentials` missing or not applied. |
| `connection() error` / TLS | Re-run `./fetch-docdb-ca-bundle.sh` and `kubectl apply -k .` so ConfigMap `docdb-ca-cert` is present. |
| `Unsupported mechanism [ -301 ]` on **docdb-init** Job | DocumentDB needs **SCRAM-SHA-1** for `mongosh` (fixed in current scripts). Run `kubectl apply -k .` to refresh `docdb-init-scripts`, delete the Job, and re-apply `docdb-init-job.yaml`. ReaperC2 itself may still run — indexes are also created on app startup. |
| `Beacon API listening` then exit | Rare; check full logs for admin/beacon bind errors. |

**Password typo (secret ≠ DocumentDB user)**

```bash
# 1. Fix examples/documentdb-secret.local.yaml (password + auth_source), then:
kubectl apply -f examples/documentdb-secret.local.yaml

# 2. Recreate app user (master creds in documentdb-admin-secret.yaml)
kubectl apply -f examples/documentdb-admin-secret.local.yaml
kubectl delete job docdb-init-user -n reaperc2-ns --ignore-not-found
kubectl apply -f docdb-init-user-job.yaml
kubectl wait -n reaperc2-ns job/docdb-init-user --for=condition=complete --timeout=120s

# 3. Restart ReaperC2
kubectl rollout restart deployment/reaperc2-deployment -n reaperc2-ns
```

If `createUser` fails because the user already exists, drop the user from DocumentDB with your infra repo / `mongosh`, then re-run the Job.

After the user Job succeeds, run the schema Job if you have not already:

```bash
kubectl apply -f docdb-init-job.yaml
kubectl wait -n reaperc2-ns job/docdb-init --for=condition=complete --timeout=120s
```

## Teardown (app only)

```bash
kubectl delete -k . --ignore-not-found
kubectl delete -f examples/documentdb-secret.local.yaml --ignore-not-found
kubectl delete -f docdb-init-job.yaml -f docdb-init-user-job.yaml --ignore-not-found
kubectl delete deployment/ollama service/ollama pvc/ollama-data -n reaperc2-ns --ignore-not-found
```
