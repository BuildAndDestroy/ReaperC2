# AWS: Deploy ReaperC2 on EKS

Kubernetes manifests for ReaperC2 on an **existing** AWS stack:

- **EKS** cluster (kubeconfig already configured)
- **ECR** image for ReaperC2
- **Amazon DocumentDB** (endpoint and credentials from your infra repo)
- **Traefik** ingress controller and load balancer

Infrastructure provisioning (VPC, EKS, DocumentDB, Traefik install) lives in a separate repository. This directory only deploys the application.

## DocumentDB pitfalls (read first)

These caused most deploy pain — avoid them up front:

| Pitfall | What to do |
|---------|------------|
| **`auth_source` ≠ `database`** | In `documentdb-secret.local.yaml`, set both to the same value (e.g. `reaperc2_db`). |
| **Secret password ≠ DocumentDB password** | Changing `.local.yaml` alone is not enough. Run [`docdb-init-user-job`](#sync-documentdb-password) — it **updates** the password if the user already exists. |
| **Only `User already exists` in logs** | Old behavior; current script prints **`Updated password for existing user`**. Run `kubectl apply -k .` then re-run the user Job. |
| **`docdb-init` Job errors** | Run `kubectl apply -k .` first (refreshes SCRAM-SHA-1 scripts), then re-apply the Job. |
| **App still uses wrong auth DB** | `deployment.yaml` reads `auth_source` from the secret. After editing the secret: `kubectl apply -f deployment.yaml` and rollout restart. |

ReaperC2 only needs DocumentDB for data — no Kubernetes PVC for the app. Operator AI uses Bedrock (see [Bedrock credentials](#bedrock-credentials-rotation)), not in-cluster Ollama.

## Quick install (script)

[`deploy-aws-k8s.sh`](deploy-aws-k8s.sh) runs the same steps as the manual checklist in order: **local prerequisites** (three `examples/*.local.yaml` files, bundled YAML, real ECR image in `deployment.yaml`, executable `fetch-docdb-ca-bundle.sh` — use `./deploy-aws-k8s.sh check-local` to verify), cluster preflight, CA bundle download, secrets (including `../operator-ai.local.yaml` if present), ECR pull secret, **`kubectl apply -k .` (core stack only — no Ingress)**, rollout, then prints commands for DocumentDB Jobs and ingress.

```bash
cd deployments/k8s/AWS
chmod +x deploy-aws-k8s.sh fetch-docdb-ca-bundle.sh
# After copying/editing all three examples/*.local.yaml and deployment image:
./deploy-aws-k8s.sh check-local
./deploy-aws-k8s.sh help
./deploy-aws-k8s.sh all
./deploy-aws-k8s.sh job-docdb-user
./deploy-aws-k8s.sh job-docdb-init   # optional
# When Traefik + Traefik CRDs + (optional) cert-manager are ready:
./deploy-aws-k8s.sh apply-ingress
```

`SKIP_ECR_SECRET=1 ./deploy-aws-k8s.sh all` skips the ECR docker-registry secret step if you manage pulls another way. `REAPER_NS` overrides the namespace. To remove the app from the cluster, see [Teardown (app only)](#teardown-app-only) (`./deploy-aws-k8s.sh teardown`).

**Why ingress is separate:** `ingress.yaml` references cert-manager and Traefik; `ingressroute.yaml` needs the Traefik CRD. Applying them with the rest of the stack often blocks or confuses first-time bring-up. Apply ingress only when those dependencies exist ([Ingress troubleshooting](#ingress-traefik-cert-manager)).

**Upgrading from an older clone** where `ingress.yaml` was in `kustomization.yaml`: `kubectl apply -k .` does not delete existing objects. If you want ingress fully managed by the script going forward, delete any old Ingress/IngressRoute once, then use `apply-ingress` for updates.

## Run from scratch (checklist)

From the **repo root**, after [`make build`](#build-and-push):

**0. Edit local files (do not commit secrets)**

```bash
cd deployments/k8s/AWS
cp examples/documentdb-secret.yaml examples/documentdb-secret.local.yaml
cp examples/documentdb-admin-secret.yaml examples/documentdb-admin-secret.local.yaml
cp examples/admin-bootstrap-secret.yaml examples/admin-bootstrap-secret.local.yaml
```

Edit:

- `examples/documentdb-secret.local.yaml` — host, `username`, `password`, `database`, **`auth_source` (same as `database`)**
- `examples/documentdb-admin-secret.local.yaml` — DocumentDB **master** user (init Job only)
- `examples/admin-bootstrap-secret.local.yaml` — first **admin UI** login (only when `operators` collection is empty)
- `deployment.yaml` — ECR `image:` tag you pushed
- `ingress.yaml` / `ingressroute.yaml` — beacon hostname

**1. TLS CA bundle** (once per clone; gitignored)

```bash
chmod +x fetch-docdb-ca-bundle.sh && ./fetch-docdb-ca-bundle.sh
```

**2. Namespace and secrets**

```bash
kubectl apply -f namespace.yaml
kubectl apply -f examples/documentdb-secret.local.yaml
kubectl apply -f examples/documentdb-admin-secret.local.yaml
kubectl apply -f examples/admin-bootstrap-secret.local.yaml

# Operator AI (ConfigMap + API keys — copy template, edit, apply; never commit .local)
cp ../operator-ai.yaml ../operator-ai.local.yaml
# Edit ../operator-ai.local.yaml (Foundry URL, deployment name, Bedrock/Foundry keys)
kubectl apply -f ../operator-ai.local.yaml

# ECR pull (replace ACCOUNT / region)
kubectl create secret docker-registry reaperc2-myregistrykey \
  --namespace=reaperc2-ns \
  --docker-server=ACCOUNT.dkr.ecr.us-east-1.amazonaws.com \
  --docker-username=AWS \
  --docker-password="$(aws ecr get-login-password --region us-east-1)" \
  --dry-run=client -o yaml | kubectl apply -f -
```

**3. App stack + DocumentDB ConfigMaps** (Ingress is **not** included — use `./deploy-aws-k8s.sh apply-ingress` later, or `kubectl apply -f ingress.yaml -f ingressroute.yaml -n reaperc2-ns`)

```bash
kubectl apply -k .
# Or: ./deploy-aws-k8s.sh apply-core
```

**4. Sync app user password to DocumentDB** (required on first deploy and after any password change)

```bash
kubectl delete job docdb-init-user -n reaperc2-ns --ignore-not-found
kubectl apply -f docdb-init-user-job.yaml
kubectl wait -n reaperc2-ns job/docdb-init-user --for=condition=complete --timeout=120s
kubectl logs -n reaperc2-ns job/docdb-init-user
```

Logs must include **`Created user`** or **`Updated password for existing user`**.

**5. Collections + indexes** (optional but recommended; idempotent)

```bash
kubectl delete job docdb-init -n reaperc2-ns --ignore-not-found
kubectl apply -f docdb-init-job.yaml
kubectl wait -n reaperc2-ns job/docdb-init --for=condition=complete --timeout=120s
kubectl logs -n reaperc2-ns job/docdb-init
```

**6. Verify**

```bash
kubectl get pods -n reaperc2-ns
kubectl logs -n reaperc2-ns deployment/reaperc2-deployment --tail=20
kubectl exec -n reaperc2-ns deployment/reaperc2-deployment -- env | grep '^MONGO_'
kubectl port-forward -n reaperc2-ns deployment/reaperc2-deployment 8443:8443
```

Expect `MONGO_DATABASE` and `MONGO_AUTH_SOURCE` to match your `documentdb-secret.local.yaml`.

### Sync DocumentDB password

Whenever you change `password` (or `username`) in `documentdb-secret.local.yaml`:

```bash
kubectl apply -f examples/documentdb-secret.local.yaml
kubectl apply -k .
kubectl delete job docdb-init-user -n reaperc2-ns --ignore-not-found
kubectl apply -f docdb-init-user-job.yaml
kubectl wait -n reaperc2-ns job/docdb-init-user --for=condition=complete --timeout=120s
kubectl logs -n reaperc2-ns job/docdb-init-user
kubectl apply -f deployment.yaml
kubectl rollout restart deployment/reaperc2-deployment -n reaperc2-ns
```

Skip the user Job only if your **infra repo** already created the user with exactly this password and `auth_source`.

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
| Admin UI first login | `examples/admin-bootstrap-secret.local.yaml` → Secret `reaperc2-admin-bootstrap` |
| ECR pull secret | `examples/registry-secret.yaml` (commands only) |
| Operator AI | [`../operator-ai.local.yaml`](../operator-ai.local.yaml) from [`../operator-ai.yaml`](../operator-ai.yaml) + IRSA on `reaperc2` for Bedrock |

## Deploy (details)

Use the [checklist](#run-from-scratch-checklist) order. Notes:

- **TLS:** `fetch-docdb-ca-bundle.sh` downloads the RDS PEM into `rds-combined-ca-bundle.pem` (gitignored). Kustomize embeds it in ConfigMap `docdb-ca-cert` on `kubectl apply -k .`. ReaperC2 sets `DEPLOY_ENV=AWS` for DocumentDB TLS ([`pkg/dbconnections/mongoconnections.go`](../../../pkg/dbconnections/mongoconnections.go)).
- **Secrets:** split keys in `documentdb-secret.local.yaml` (`host`, `username`, `password`, `database`, `auth_source`) — not a single `mongodb-uri`. See [`examples/README.md`](examples/README.md).
- **User Job:** creates the app user or **updates its password** to match the secret. Required unless infra already created the user with the exact same password and auth database.
- **Init Job:** idempotent collections/indexes for `clients`, `heartbeat`, `data`, operators, engagements, etc. ReaperC2 also creates admin indexes on startup if you skip this Job.

`kubectl apply -k .` applies the ReaperC2 Deployment, ServiceAccount, Service, and DocumentDB-related ConfigMaps (CA bundle + init scripts). It does **not** apply `ingress.yaml` / `ingressroute.yaml` — use [`./deploy-aws-k8s.sh apply-ingress`](deploy-aws-k8s.sh) or `kubectl apply -f ingress.yaml -f ingressroute.yaml -n reaperc2-ns` after Traefik (and cert-manager, if you use ACME annotations) are ready.

**Operator AI:** copy [`../operator-ai.yaml`](../operator-ai.yaml) → `../operator-ai.local.yaml`, set ConfigMap (Bedrock region, Foundry URL, **Azure deployment names**) and Secret (API keys). Apply the **`.local`** file only:

```bash
kubectl apply -f ../operator-ai.local.yaml
kubectl rollout restart deployment/reaperc2-deployment -n reaperc2-ns
```

Bedrock on EKS: **IRSA** ([`examples/bedrock-irsa.md`](examples/bedrock-irsa.md)) or Bedrock API key in the Secret. Foundry: `REAPER_AI_FOUNDRY_API_KEY` + resource URL (`https://YOUR_RESOURCE.openai.azure.com`) + deployment name from `az cognitiveservices account deployment list`.

Raw `kubectl apply -k .` does **not** apply Operator AI (your live AI config stays in `.local` files, not in kustomize). The [`deploy-aws-k8s.sh`](deploy-aws-k8s.sh) `apply-secrets` and `all` commands apply `../operator-ai.local.yaml` when that file exists.

**Admin UI login** is **not** DocumentDB. On first boot (empty `operators` collection), use `username` / `password` from `reaperc2-admin-bootstrap`. Change the password after login or create operators in MongoDB and remove the bootstrap secret.

```bash
kubectl create secret generic reaperc2-ai-secrets -n reaperc2-ns \
  --from-literal=REAPER_AI_BEDROCK_ACCESS_KEY_ID=... \
  --from-literal=REAPER_AI_BEDROCK_SECRET_ACCESS_KEY=... \
  --from-literal=REAPER_AI_BEDROCK_SESSION_TOKEN=... \
  --from-literal=REAPER_AI_OPENAI_API_KEY=sk-... \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl rollout restart deployment/reaperc2-deployment -n reaperc2-ns
```

See [Operator AI](../../../docs/operator-guide-ai.md) for Bedrock inference profile IDs and IAM.

### Bedrock credentials (rotation)

ReaperC2 reads Bedrock creds from Secret **`reaperc2-ai-secrets`** at **pod start**. Updating the Secret alone does not refresh running pods — you must **roll out** after every credential change.

| What you use | Keys in `reaperc2-ai-secrets` |
|--------------|-------------------------------|
| **Bedrock API key** (console “Generate API key”) | `REAPER_AI_BEDROCK_API_KEY` |
| **Temporary IAM** (SSO / STS, often ~12h) | `REAPER_AI_BEDROCK_ACCESS_KEY_ID`, `REAPER_AI_BEDROCK_SECRET_ACCESS_KEY`, **`REAPER_AI_BEDROCK_SESSION_TOKEN`** |
| **Long-lived IAM user** | access key + secret (no session token) |

**When keys rotate (e.g. every 12 hours):**

```bash
# Temporary IAM (include session token) or a new Bedrock API key:
kubectl create secret generic reaperc2-ai-secrets -n reaperc2-ns \
  --from-literal=REAPER_AI_BEDROCK_ACCESS_KEY_ID=AKIA... \
  --from-literal=REAPER_AI_BEDROCK_SECRET_ACCESS_KEY=... \
  --from-literal=REAPER_AI_BEDROCK_SESSION_TOKEN=... \
  --dry-run=client -o yaml | kubectl apply -f -

# Or Bedrock API key only:
# --from-literal=REAPER_AI_BEDROCK_API_KEY='...'

kubectl rollout restart deployment/reaperc2-deployment -n reaperc2-ns
kubectl rollout status deployment/reaperc2-deployment -n reaperc2-ns
```

**Recommended on EKS — IRSA (no manual 12h updates):** attach a Bedrock-capable IAM role to the ReaperC2 ServiceAccount, set in `operator-ai.local.yaml` ConfigMap:

```yaml
REAPER_AI_BEDROCK_USE_IAM: "1"
```

Remove Bedrock keys from `reaperc2-ai-secrets`, `kubectl apply -k .`, and restart once. The AWS SDK uses the pod role; EKS refreshes the web identity token automatically.

Step-by-step IRSA setup: [`examples/bedrock-irsa.md`](examples/bedrock-irsa.md). Example IAM policy: [`examples/bedrock-iam-policy.json`](examples/bedrock-iam-policy.json).

**Troubleshooting — `AccessDeniedException` on `bedrock:InvokeModel` with the EKS *node group* role in the error:** the pod is using the **node instance profile**, not IRSA. Fix: create the Bedrock policy + `eksctl create iamserviceaccount` (or annotate `serviceaccount.yaml` with `eks.amazonaws.com/role-arn`), ensure `REAPER_AI_BEDROCK_USE_IAM=1`, rollout restart, and confirm `AWS_ROLE_ARN` is set in the pod. Alternatively, put a Bedrock API key in `reaperc2-ai-secrets` (see [`operator-ai.yaml`](../operator-ai.yaml)).

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
| `kustomization.yaml` | `kubectl apply -k` entrypoint (core app + DocumentDB ConfigMaps; **no** ingress) |
| `deploy-aws-k8s.sh` | Install/update: `check-local`, `all`, Jobs, `apply-ingress`, `teardown-ingress`, `teardown` |
| `namespace.yaml` | `reaperc2-ns` |
| [`../operator-ai.yaml`](../operator-ai.yaml) | Operator AI template (ConfigMap + Secret); apply `operator-ai.local.yaml` |
| `deployment.yaml` | ReaperC2 + DocumentDB env + CA volume |
| `service.yaml` | ClusterIP :8080 (beacon) |
| `ingress.yaml` | Standard Ingress for Traefik / cert-manager (**apply after** Traefik + cert-manager) |
| `ingressroute.yaml` | Traefik `IngressRoute` (beacon :8080) — needs CRD `ingressroutes.traefik.io` |
| `fetch-docdb-ca-bundle.sh` | Downloads RDS global CA PEM |
| `docdb-init-job.yaml` | One-shot Job: collections + indexes |
| `docdb-init-user-job.yaml` | Create/update app DB user password (master creds) |
| `scripts/docdb-init.js` | Idempotent schema script (used by init Job) |
| `examples/documentdb-secret.yaml` | DocumentDB secret **template** (placeholders) |
| `examples/documentdb-secret.local.yaml` | Your real secret (gitignored) |
| `examples/documentdb-admin-secret.yaml` | Admin secret **template** for init-user Job |
| `examples/README.md` | How to copy templates → `.local.yaml` |
| `examples/registry-secret.yaml` | ECR pull secret instructions |

## Troubleshooting

### Ingress, Traefik, cert-manager

| Symptom | What to check |
|---------|----------------|
| `apply-ingress` fails: no matches for kind `IngressRoute` | Install Traefik **with CRDs** in the cluster (or temporarily skip `ingressroute.yaml` if you only use standard Ingress). |
| `apply-ingress` fails: `clusterissuer` / cert-manager | Install cert-manager and create `ClusterIssuer` **`letsencrypt-prod`** (or edit `ingress.yaml` annotations to match your issuer name). |
| Ingress stuck, HTTP-01 challenges fail | DNS for the hostname must point at your Traefik LB; security groups must allow HTTP (80) from the internet for ACME. |
| App works via `port-forward` but not via ingress | Confirm `IngressClass` name is **`traefik`** (or change `ingressClassName` in `ingress.yaml`). Run `./deploy-aws-k8s.sh preflight`. |
| Switch Let's Encrypt **staging → prod** (or change issuer) | Run `./deploy-aws-k8s.sh teardown-ingress` to remove Ingress/IngressRoute, the TLS Secret (`tls.secretName` in `ingress.yaml`), and cert-manager `Certificate` resources, then edit `cert-manager.io/cluster-issuer` in `ingress.yaml` and run `./deploy-aws-k8s.sh apply-ingress`. Does not remove the app `Deployment`. |

Until ingress is sorted, use **`kubectl port-forward`** for the admin UI and (if needed) a separate path for beacon traffic.

### `reaperc2-deployment` CrashLoopBackOff

Confirm the error (almost always MongoDB on startup):

```bash
kubectl logs -n reaperc2-ns deployment/reaperc2-deployment --previous --tail=30
```

| Log line | Fix |
|----------|-----|
| `MongoDB Ping Error` / `authentication failed` | See [DocumentDB pitfalls](#documentdb-pitfalls-read-first) and [Sync DocumentDB password](#sync-documentdb-password). Usually `auth_source` ≠ `database` or secret password not synced to DocumentDB. |
| `MONGO_HOST` / `MONGO_PASSWORD` required | Secret `reaperc2-documentdb-credentials` missing or not applied. |
| `connection() error` / TLS | Re-run `./fetch-docdb-ca-bundle.sh` and `kubectl apply -k .` so ConfigMap `docdb-ca-cert` is present. |
| `Unsupported mechanism [ -301 ]` on **docdb-init** Job | DocumentDB needs **SCRAM-SHA-1** for `mongosh` (fixed in current scripts). Run `kubectl apply -k .` to refresh `docdb-init-scripts`, delete the Job, and re-apply `docdb-init-job.yaml`. ReaperC2 itself may still run — indexes are also created on app startup. |
| `Beacon API listening` then exit | Rare; check full logs for admin/beacon bind errors. |

**Auth still failing?** Run the full [Sync DocumentDB password](#sync-documentdb-password) block and confirm pod env: `kubectl exec -n reaperc2-ns deployment/reaperc2-deployment -- env | grep '^MONGO_'`.

If you applied manifests without `-n reaperc2-ns` and the YAML had no `metadata.namespace`, resources can land in **`default`**. Inspect then delete (run locally where `kubectl` + AWS auth work):

```bash
NS=default
kubectl get deploy,svc,po,ing,job,cm,secret,sa -n "$NS"
kubectl get ingressroute -n "$NS" 2>/dev/null || true

kubectl delete deployment reaperc2-deployment -n "$NS" --ignore-not-found
kubectl delete svc reaperc2-service -n "$NS" --ignore-not-found
kubectl delete ingress reaperc2-ingress -n "$NS" --ignore-not-found
kubectl delete ingressroute reaperc2-ingressroute -n "$NS" --ignore-not-found
kubectl delete job docdb-init docdb-init-user -n "$NS" --ignore-not-found
kubectl delete sa reaperc2 -n "$NS" --ignore-not-found
kubectl delete cm docdb-ca-cert docdb-init-scripts reaperc2-ai-config -n "$NS" --ignore-not-found
kubectl delete secret reaperc2-documentdb-credentials reaperc2-documentdb-admin reaperc2-admin-bootstrap reaperc2-ai-secrets reaperc2-myregistrykey -n "$NS" --ignore-not-found
```

If you used cert-manager ACME on that ingress, also `kubectl get certificate,certificaterequest -n "$NS"` and delete any Reaper-related certificates.

## Teardown (app only)

Removes the ReaperC2 workload, ingress, DocumentDB app secret (if the `.local.yaml` file exists), init Jobs, and legacy Ollama objects. It does **not** delete the namespace, DocumentDB itself, admin/bootstrap secrets, ECR pull secret, or Operator AI — remove those with `kubectl delete` if you want a clean namespace.

```bash
cd deployments/k8s/AWS
./deploy-aws-k8s.sh teardown
```

Equivalent manual commands:

```bash
kubectl delete -k . --ignore-not-found
kubectl delete -f ingress.yaml -f ingressroute.yaml -n reaperc2-ns --ignore-not-found
kubectl delete -f examples/documentdb-secret.local.yaml --ignore-not-found
kubectl delete -f docdb-init-job.yaml -f docdb-init-user-job.yaml --ignore-not-found
kubectl delete deployment/ollama service/ollama pvc/ollama-data -n reaperc2-ns --ignore-not-found
```
