# Deploy ReaperC2 on Kubernetes (EKS or k3s)

Kubernetes manifests and [`deploy-cluster.sh`](deploy-cluster.sh) for ReaperC2 against **Amazon DocumentDB** (same app and TLS bundle whether the cluster is EKS, k3s, or another distro):

- **Kubeconfig** pointed at your cluster
- **Container image** — typically **ECR** on AWS (`overlays/aws-ecr` adds `imagePullSecrets`); on **k3s** use `REAPER_CLUSTER=k3s` and any registry (public, in-cluster, or opt-in ECR secret)
- **DocumentDB** endpoint and credentials
- **Traefik** + **cert-manager** (same Ingress + IngressRoute pattern as EKS)

Infrastructure provisioning (VPC, cluster, DocumentDB, Traefik) lives in a separate repository. This directory only deploys the application.

**Legacy path:** `kubectl apply -k deployments/k8s/AWS` still resolves to the **aws-ecr** overlay. Prefer working from this directory (`deployments/k8s/reaperc2`).

## DocumentDB pitfalls (read first)

These caused most deploy pain — avoid them up front:

| Pitfall | What to do |
|---------|------------|
| **`auth_source` ≠ `database`** | In `documentdb-secret.local.yaml`, set both to the same value (e.g. `reaperc2_db`). |
| **Secret password ≠ DocumentDB password** | Changing `.local.yaml` alone is not enough. Run [`docdb-init-user-job`](#sync-documentdb-password) — it **updates** the password if the user already exists. |
| **Only `User already exists` in logs** | Old behavior; current script prints **`Updated password for existing user`**. Run `./deploy-cluster.sh apply-core` then re-run the user Job. |
| **`docdb-init` Job errors** | Run `./deploy-cluster.sh apply-core` first (refreshes SCRAM-SHA-1 scripts), then re-apply the Job. |
| **App still uses wrong auth DB** | `base/deployment.yaml` reads `auth_source` from the secret. After editing the secret: `kubectl apply -f base/deployment.yaml` and rollout restart. |

ReaperC2 only needs DocumentDB for data — no Kubernetes PVC for the app. Operator AI uses Bedrock (see [Bedrock credentials](#bedrock-credentials-rotation)), not in-cluster Ollama.

## Quick install (script)

[`deploy-cluster.sh`](deploy-cluster.sh) runs the same steps as the manual checklist in order: **local prerequisites** (three `examples/*.local.yaml` files, bundled YAML under `base/`, real ECR image in `base/deployment.yaml` when `REAPER_CLUSTER=aws`, executable `base/fetch-docdb-ca-bundle.sh` — use `./deploy-cluster.sh check-local` to verify), cluster preflight, CA bundle download, secrets (including `../operator-ai.local.yaml` if present), ECR pull secret (aws profile, or opt-in on k3s), **`kubectl apply -k` on the active overlay** (core stack only — no Ingress), rollout, then prints commands for DocumentDB Jobs and ingress.

```bash
cd deployments/k8s/reaperc2
chmod +x deploy-cluster.sh base/fetch-docdb-ca-bundle.sh
# EKS / ECR (default REAPER_CLUSTER=aws):
./deploy-cluster.sh check-local
./deploy-cluster.sh help
./deploy-cluster.sh all
./deploy-cluster.sh job-docdb-user
./deploy-cluster.sh job-docdb-init   # optional
./deploy-cluster.sh apply-ingress

# k3s (no ECR imagePullSecret unless REAPER_ECR_SECRET=1):
REAPER_CLUSTER=k3s ./deploy-cluster.sh check-local
REAPER_CLUSTER=k3s ./deploy-cluster.sh all
REAPER_CLUSTER=k3s ./deploy-cluster.sh apply-ingress
```

`SKIP_ECR_SECRET=1 ./deploy-cluster.sh all` skips the ECR docker-registry secret on **aws**. On **k3s**, `all` skips `ecr-secret` by default; set `REAPER_ECR_SECRET=1` if you still pull from ECR. `REAPER_NS` overrides the namespace. Teardown: `./deploy-cluster.sh teardown` (respects `REAPER_CLUSTER` for the overlay).

**Why ingress is separate:** `ingress.yaml` references cert-manager and Traefik; `ingressroute.yaml` needs the Traefik CRD. Applying them with the rest of the stack often blocks or confuses first-time bring-up. Apply ingress only when those dependencies exist ([Ingress troubleshooting](#ingress-traefik-cert-manager)).

**Upgrading from an older clone** where `ingress.yaml` was in `kustomization.yaml`: re-applying the overlay does not delete existing objects. If you want ingress fully managed by the script going forward, delete any old Ingress/IngressRoute once, then use `apply-ingress` for updates.

### k3s notes

Use the **same** Traefik IngressClass (`traefik`), cert-manager, `ingress.yaml`, and `ingressroute.yaml` as on EKS.

- Set **`REAPER_CLUSTER=k3s`** for `check-local`, `all`, `apply-core`, `apply-ingress`, `teardown`, etc. The **`overlays/k3s`** stack matches **`overlays/aws-ecr`** except it does **not** add `imagePullSecrets` (use a public image, a registry your nodes trust, or run `REAPER_ECR_SECRET=1 ./deploy-cluster.sh ecr-secret` after `apply-secrets`).
- Keep **`DEPLOY_ENV=AWS`** in `base/deployment.yaml` when connecting to **DocumentDB** so TLS/query options match the AWS driver path ([`pkg/dbconnections/mongoconnections.go`](../../../pkg/dbconnections/mongoconnections.go)).
- **Bedrock / IAM:** k3s has no EKS IRSA. Prefer **API keys** in `../operator-ai.local.yaml` / `reaperc2-ai-secrets` (see [`examples/bedrock-irsa.md`](examples/bedrock-irsa.md) for the EKS-only role flow).

## Run from scratch (checklist)

From the **repo root**, after [`make build`](#build-and-push):

**0. Edit local files (do not commit secrets)**

```bash
cd deployments/k8s/reaperc2
cp examples/documentdb-secret.yaml examples/documentdb-secret.local.yaml
cp examples/documentdb-admin-secret.yaml examples/documentdb-admin-secret.local.yaml
cp examples/admin-bootstrap-secret.yaml examples/admin-bootstrap-secret.local.yaml
```

Edit:

- `examples/documentdb-secret.local.yaml` — host, `username`, `password`, `database`, **`auth_source` (same as `database`)**
- `examples/documentdb-admin-secret.local.yaml` — DocumentDB **master** user (init Job only)
- `examples/admin-bootstrap-secret.local.yaml` — first **admin UI** login (only when `operators` collection is empty)
- `base/deployment.yaml` — container `image:` (ECR on aws profile; any registry on k3s)
- `ingress.yaml` / `ingressroute.yaml` — beacon hostname

**1. TLS CA bundle** (once per clone; gitignored)

```bash
chmod +x base/fetch-docdb-ca-bundle.sh && ./deploy-cluster.sh fetch-ca
```

**2. Namespace and secrets**

```bash
kubectl apply -f base/namespace.yaml
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

**3. App stack + DocumentDB ConfigMaps** (Ingress is **not** included — use `./deploy-cluster.sh apply-ingress` later, or `kubectl apply -f ingress.yaml -f ingressroute.yaml -n reaperc2-ns`)

```bash
kubectl apply -k overlays/aws-ecr
# Or: ./deploy-cluster.sh apply-core   # default REAPER_CLUSTER=aws
# k3s: REAPER_CLUSTER=k3s ./deploy-cluster.sh apply-core
```

**4. Sync app user password to DocumentDB** (required on first deploy and after any password change)

```bash
kubectl delete job docdb-init-user -n reaperc2-ns --ignore-not-found
kubectl apply -f base/docdb-init-user-job.yaml
kubectl wait -n reaperc2-ns job/docdb-init-user --for=condition=complete --timeout=120s
kubectl logs -n reaperc2-ns job/docdb-init-user
```

Logs must include **`Created user`** or **`Updated password for existing user`**.

**5. Collections + indexes** (optional but recommended; idempotent)

```bash
kubectl delete job docdb-init -n reaperc2-ns --ignore-not-found
kubectl apply -f base/docdb-init-job.yaml
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
./deploy-cluster.sh apply-core
kubectl delete job docdb-init-user -n reaperc2-ns --ignore-not-found
kubectl apply -f base/docdb-init-user-job.yaml
kubectl wait -n reaperc2-ns job/docdb-init-user --for=condition=complete --timeout=120s
kubectl logs -n reaperc2-ns job/docdb-init-user
kubectl apply -f base/deployment.yaml
kubectl rollout restart deployment/reaperc2-deployment -n reaperc2-ns
```

Skip the user Job only if your **infra repo** already created the user with exactly this password and `auth_source`.

## Prerequisites

- `kubectl` pointed at your cluster (`aws eks update-kubeconfig ...` for EKS, or your k3s kubeconfig)
- ReaperC2 image built and pushed to ECR (see [Build and push](#build-and-push) below)
- DocumentDB cluster endpoint and application DB user (`api_user` / `api_db` or your naming)
- Traefik installed with an `IngressClass` named `traefik` (adjust manifests if yours differs)

## Build and push

From the repo root, build **linux/amd64** and **linux/arm64** and push a multi-arch manifest to ECR (defaults match `base/deployment.yaml`; override with env vars):

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

Then set `base/deployment.yaml` `image:` to the tag you pushed (e.g. `123456789012.dkr.ecr.us-east-1.amazonaws.com/reaperc2:abc1234` — replace with your AWS account ID).

## Configure before apply

| Item | File |
|------|------|
| ECR image URI | `base/deployment.yaml` → `spec.template.spec.containers[0].image` |
| Beacon hostname / TLS | `ingress.yaml`, `ingressroute.yaml` → `subdomain.domain.com` |
| DocumentDB host, user, password | `examples/documentdb-secret.local.yaml` (from template; not committed) |
| Admin UI first login | `examples/admin-bootstrap-secret.local.yaml` → Secret `reaperc2-admin-bootstrap` |
| ECR pull secret | `examples/registry-secret.yaml` (commands only) |
| Operator AI | [`../operator-ai.local.yaml`](../operator-ai.local.yaml) from [`../operator-ai.yaml`](../operator-ai.yaml) + IRSA on `reaperc2` for Bedrock |

## Deploy (details)

Use the [checklist](#run-from-scratch-checklist) order. Notes:

- **TLS:** `base/fetch-docdb-ca-bundle.sh` (or `./deploy-cluster.sh fetch-ca`) downloads the RDS PEM into `base/rds-combined-ca-bundle.pem` (gitignored). Kustomize embeds it in ConfigMap `docdb-ca-cert` when you run `./deploy-cluster.sh apply-core` (or `kubectl apply -k overlays/aws-ecr` / `overlays/k3s`). ReaperC2 sets `DEPLOY_ENV=AWS` for DocumentDB TLS ([`pkg/dbconnections/mongoconnections.go`](../../../pkg/dbconnections/mongoconnections.go)).
- **Secrets:** split keys in `documentdb-secret.local.yaml` (`host`, `username`, `password`, `database`, `auth_source`) — not a single `mongodb-uri`. See [`examples/README.md`](examples/README.md).
- **User Job:** creates the app user or **updates its password** to match the secret. Required unless infra already created the user with the exact same password and auth database.
- **Init Job:** idempotent collections/indexes for `clients`, `heartbeat`, `data`, operators, engagements, etc. ReaperC2 also creates admin indexes on startup if you skip this Job.

`./deploy-cluster.sh apply-core` applies the ReaperC2 Deployment, ServiceAccount, Service, and DocumentDB-related ConfigMaps (CA bundle + init scripts). It does **not** apply `ingress.yaml` / `ingressroute.yaml` — use [`./deploy-cluster.sh apply-ingress`](deploy-cluster.sh) or `kubectl apply -f ingress.yaml -f ingressroute.yaml -n reaperc2-ns` after Traefik (and cert-manager, if you use ACME annotations) are ready.

**Operator AI:** copy [`../operator-ai.yaml`](../operator-ai.yaml) → `../operator-ai.local.yaml`, set ConfigMap (Bedrock region, Foundry URL, **Azure deployment names**) and Secret (API keys). Apply the **`.local`** file only:

```bash
kubectl apply -f ../operator-ai.local.yaml
kubectl rollout restart deployment/reaperc2-deployment -n reaperc2-ns
```

Bedrock on EKS: **IRSA** ([`examples/bedrock-irsa.md`](examples/bedrock-irsa.md)) or Bedrock API key in the Secret. Foundry: `REAPER_AI_FOUNDRY_API_KEY` + resource URL (`https://YOUR_RESOURCE.openai.azure.com`) + deployment name from `az cognitiveservices account deployment list`.

Raw `kubectl apply -k overlays/...` does **not** apply Operator AI (your live AI config stays in `.local` files, not in kustomize). The [`deploy-cluster.sh`](deploy-cluster.sh) `apply-secrets` and `all` commands apply `../operator-ai.local.yaml` when that file exists.

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

Remove Bedrock keys from `reaperc2-ai-secrets`, `./deploy-cluster.sh apply-core`, and restart once. The AWS SDK uses the pod role; EKS refreshes the web identity token automatically.

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

| Path | Purpose |
|------|---------|
| `deploy-cluster.sh` | Install/update: `check-local`, `all`, Jobs, `apply-ingress`, `teardown-ingress`, `teardown` (`REAPER_CLUSTER=aws` or `k3s`) |
| `overlays/aws-ecr/` | Kustomize overlay: `base` + ECR `imagePullSecrets` |
| `overlays/k3s/` | Kustomize overlay: `base` only (no registry secret patch) |
| `base/kustomization.yaml` | `kubectl apply -k` entrypoint for **base** (included by overlays; **no** ingress) |
| `base/namespace.yaml` | `reaperc2-ns` |
| [`../operator-ai.yaml`](../operator-ai.yaml) | Operator AI template (ConfigMap + Secret); apply `operator-ai.local.yaml` |
| `base/deployment.yaml` | ReaperC2 + DocumentDB env + CA volume |
| `base/service.yaml` | ClusterIP :8080 (beacon) |
| `ingress.yaml` | Standard Ingress for Traefik / cert-manager (**apply after** Traefik + cert-manager) |
| `ingressroute.yaml` | Traefik `IngressRoute` (beacon :8080) — needs CRD `ingressroutes.traefik.io` |
| `base/fetch-docdb-ca-bundle.sh` | Downloads RDS global CA PEM into `base/` |
| `base/docdb-init-job.yaml` | One-shot Job: collections + indexes |
| `base/docdb-init-user-job.yaml` | Create/update app DB user password (master creds) |
| `base/scripts/docdb-init.js` | Idempotent schema script (used by init Job) |
| `examples/documentdb-secret.yaml` | DocumentDB secret **template** (placeholders) |
| `examples/documentdb-secret.local.yaml` | Your real secret (gitignored) |
| `examples/documentdb-admin-secret.yaml` | Admin secret **template** for init-user Job |
| `examples/README.md` | How to copy templates → `.local.yaml` |
| `examples/registry-secret.yaml` | ECR pull secret instructions |

## Troubleshooting

### Beacon TLS (Scythe)

| Symptom | What to check |
|---------|----------------|
| **`x509: certificate signed by unknown authority`** on `GET https://…/heartbeat` | Usually **not** the ingress YAML once the public chain is valid — the **beacon host** is seeing a **different** cert (split DNS, proxy, or stale Scythe build). See [Beacon troubleshooting](../../../docs/operator-guide-beacons.md#beacon-troubleshooting). |

### Ingress, Traefik, cert-manager

| Symptom | What to check |
|---------|----------------|
| `apply-ingress` fails: no matches for kind `IngressRoute` | Install Traefik **with CRDs** in the cluster (or temporarily skip `ingressroute.yaml` if you only use standard Ingress). |
| `apply-ingress` fails: `clusterissuer` / cert-manager | Install cert-manager and create `ClusterIssuer` **`letsencrypt-prod`** (or edit `ingress.yaml` annotations to match your issuer name). |
| Ingress stuck, HTTP-01 challenges fail | DNS for the hostname must point at your Traefik LB; security groups must allow HTTP (80) from the internet for ACME. |
| App works via `port-forward` but not via ingress | Confirm `IngressClass` name is **`traefik`** (or change `ingressClassName` in `ingress.yaml`). Run `./deploy-cluster.sh preflight`. |
| **cert-manager + Traefik** look “stuck” or wrong cert | Do **not** use `tls.certResolver` on `IngressRoute` when the `Ingress` uses cert-manager — use `tls.secretName` matching `ingress.yaml` `tls.secretName` so one Secret is issued once. |
| Switch Let's Encrypt **staging → prod** (or change issuer) | Run `./deploy-cluster.sh teardown-ingress` to remove Ingress/IngressRoute, the TLS Secret (`tls.secretName` in `ingress.yaml`), and cert-manager `Certificate` resources, then edit `cert-manager.io/cluster-issuer` in `ingress.yaml` and run `./deploy-cluster.sh apply-ingress`. Does not remove the app `Deployment`. |

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
| `connection() error` / TLS | Re-run `./deploy-cluster.sh fetch-ca` and `./deploy-cluster.sh apply-core` so ConfigMap `docdb-ca-cert` is present. |
| `Unsupported mechanism [ -301 ]` on **docdb-init** Job | DocumentDB needs **SCRAM-SHA-1** for `mongosh` (fixed in current scripts). Run `./deploy-cluster.sh apply-core` to refresh `docdb-init-scripts`, delete the Job, and re-apply `base/docdb-init-job.yaml`. ReaperC2 itself may still run — indexes are also created on app startup. |
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
cd deployments/k8s/reaperc2
./deploy-cluster.sh teardown
```

Equivalent manual commands (delete **only** the overlay you applied — `aws-ecr` or `k3s`):

```bash
kubectl delete -k overlays/aws-ecr --ignore-not-found
# or: kubectl delete -k overlays/k3s --ignore-not-found
kubectl delete -f ingress.yaml -f ingressroute.yaml -n reaperc2-ns --ignore-not-found
kubectl delete -f examples/documentdb-secret.local.yaml --ignore-not-found
kubectl delete -f base/docdb-init-job.yaml -f base/docdb-init-user-job.yaml --ignore-not-found
kubectl delete deployment/ollama service/ollama pvc/ollama-data -n reaperc2-ns --ignore-not-found
```
