# k3s OnPrem (NFS Mongo + Ollama)

Deploy ReaperC2 on **k3s** with **NFS MongoDB**, a **private registry** image, and in-cluster **Ollama** for Operator AI.

> **Not** the DocumentDB path. For k3s + **Amazon DocumentDB**, use [`../k8s/reaperc2/`](../k8s/reaperc2/) with `REAPER_CLUSTER=k3s` ([`../k8s/k3s/README.md`](../k8s/k3s/README.md)).

## Prerequisites

- k3s (`curl -sfL https://get.k3s.io | sh -`)
- `kubectl` or `sudo k3s kubectl`
- NFS export for MongoDB; `nfs-common` on nodes
- Go on the build machine (for `build-push-image.sh`)
- ≥ 4 GiB RAM recommended for Ollama

## Quick start

```bash
cp deployments/k3s/config.example.env deployments/k3s/config.env
# Edit config.env, mongo-secret.yaml, k8s/operator-ai.yaml (or operator-ai.local.yaml)

./deployments/k3s/scripts/build-push-image.sh    # arm64 default (Pi)
./deployments/k3s/scripts/deploy.sh

kubectl port-forward -n reaperc2-ns deployment/reaperc2-deployment 8443:8443
# http://127.0.0.1:8443/ai
```

## Scripts

| Script | Purpose |
|--------|---------|
| `scripts/build-push-image.sh` | Host cross-compile + push (`--arch arm64\|amd64`, `--import-local`, `--load`) |
| `scripts/deploy.sh` | Ollama + Operator AI + Mongo + ReaperC2 |
| `scripts/seed-mongo.sh` | Seed `api_db` via port-forward |
| `scripts/reroll.sh` | Rebuild image + rollout (`--with-operator-ai`, `--fresh-mongo`) |
| `scripts/undeploy.sh` | Tear down (`--delete-pvc`) |

```bash
./deployments/k3s/scripts/deploy.sh --operator-ai-only
./deployments/k3s/scripts/reroll.sh --with-operator-ai
```

## Shared manifests

| File | Purpose |
|------|---------|
| [`../k8s/ollama.yaml`](../k8s/ollama.yaml) | In-cluster Ollama Deployment |
| [`../k8s/operator-ai.yaml`](../k8s/operator-ai.yaml) | Operator AI ConfigMap + Secret |

## Config reference

See [`../README.md`](../README.md) for `config.env` variables.

## Undeploy

```bash
./deployments/k3s/scripts/undeploy.sh
./deployments/k3s/scripts/undeploy.sh --delete-pvc   # destructive
```
