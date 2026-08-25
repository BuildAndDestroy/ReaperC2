# k3s OnPrem (NFS Mongo + Ollama)

Deploy ReaperC2 on **k3s** with **NFS MongoDB**, a **private registry** image, and in-cluster **Ollama** for Operator AI.

> **Not** the DocumentDB path. For k3s + **Amazon DocumentDB**, use [`../k8s/reaperc2/`](../k8s/reaperc2/) with `REAPER_CLUSTER=k3s` ([`../k8s/k3s/README.md`](../k8s/k3s/README.md)).

## Hostnames

| Surface | Access |
|---------|--------|
| **Beacons** | Public HTTPS on `INGRESS_HOST` (default `beacons.example.com`) → service `:8080` (`deploy.sh --with-ingress`) |
| **Admin UI** | **Port-forward only** (same as AWS reaperc2 path — not on public ingress) |
| **Operator AI** | `/ai` after admin port-forward |

## Prerequisites

- k3s (`curl -sfL https://get.k3s.io | sh -`)
- `kubectl` or `sudo k3s kubectl`
- NFS export for MongoDB (e.g. `/export/reaperc2-metric` on your NAS); `nfs-common` on nodes
- Pin MongoDB to a capable node if needed (`MONGO_NODE_HOSTNAME=control-plane`; mongo:7 needs ARMv8.2 on some ARM boards)
- Go on the build machine (for `build-push-image.sh`)
- ≥ 4 GiB RAM recommended for Ollama
- For public beacons: Traefik + cert-manager with ClusterIssuer `letsencrypt-prod` (or set `CERT_MANAGER_ISSUER` in `config.env`)
- DNS for `INGRESS_HOST` → cluster ingress

## Quick start

```bash
cp deployments/k3s/config.example.env deployments/k3s/config.env
# Edit config.env, mongo-secret.yaml, admin-bootstrap-secret.yaml, k8s/operator-ai.yaml (or operator-ai.local.yaml)

./deployments/k3s/scripts/build-push-image.sh    # arm64 default (Pi)
./deployments/k3s/scripts/deploy.sh --with-ingress

# Admin UI (never on ingress):
kubectl port-forward -n reaperc2-ns deployment/reaperc2-deployment 8443:8443
# https://127.0.0.1:8443/login — default first operator: admin / changeme (admin-bootstrap-secret.yaml)
# http://127.0.0.1:8443/ai
```

## MongoDB

- Database name: **`reaperc2-metric`** (`MONGO_DATABASE` in `config.env`)
- NFS path example: `/export/reaperc2-metric` (edit `MONGO_NFS_*` in `config.env`)
- First install requires an **empty** NFS export; app user is bootstrapped on first pod start
- Do not change `mongo-secret.yaml` passwords after Mongo has initialized without resetting NFS data

## Scripts

| Script | Purpose |
|--------|---------|
| `scripts/build-push-image.sh` | Host cross-compile + push (`--arch arm64\|amd64`, `--import-local`, `--load`) |
| `scripts/deploy.sh` | Ollama + Operator AI + Mongo + ReaperC2; `--with-ingress` for beacon TLS |
| `scripts/seed-mongo.sh` | Seed `reaperc2-metric` via port-forward |
| `scripts/reroll.sh` | Rebuild image + rollout (`--with-operator-ai`, `--fresh-mongo`, `--with-ingress`) |
| `scripts/undeploy.sh` | Tear down (`--delete-pvc`) |
| `scripts/verify-mongo-app-user.sh` | Check app user auth against `reaperc2-metric` |

```bash
./deployments/k3s/scripts/deploy.sh --operator-ai-only
./deployments/k3s/scripts/reroll.sh --with-operator-ai
./deployments/k3s/scripts/reroll.sh --fresh-mongo --with-ingress
```

## Shared manifests

| File | Purpose |
|------|---------|
| [`../k8s/ollama.yaml`](../k8s/ollama.yaml) | In-cluster Ollama Deployment (models on NFS via `OLLAMA_NFS_*` in `config.env`) |
| [`../k8s/operator-ai.yaml`](../k8s/operator-ai.yaml) | Operator AI ConfigMap + Secret |
| `ingress.yaml` / `ingressroute.yaml` | Beacon `:8080` only (matches AWS reaperc2 ingress pattern) |

## Config reference

See [`../README.md`](../README.md) for `config.env` variables.

## Undeploy

```bash
./deployments/k3s/scripts/undeploy.sh
./deployments/k3s/scripts/undeploy.sh --delete-pvc   # destructive — wipe NFS export before redeploy
```
