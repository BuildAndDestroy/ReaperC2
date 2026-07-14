# Kubernetes

Deployment layouts live under [`deployments/`](https://github.com/BuildAndDestroy/ReaperC2/tree/main/deployments/). See **[`deployments/README.md`](https://github.com/BuildAndDestroy/ReaperC2/blob/main/deployments/README.md)** for which path to use.

| Path | Use case |
|------|----------|
| [`deployments/k8s/reaperc2/`](https://github.com/BuildAndDestroy/ReaperC2/tree/main/deployments/k8s/reaperc2/) | **DocumentDB** on EKS or k3s — [`DEPLOY.md`](https://github.com/BuildAndDestroy/ReaperC2/blob/main/deployments/k8s/DEPLOY.md) |
| [`deployments/k3s/`](https://github.com/BuildAndDestroy/ReaperC2/tree/main/deployments/k3s/) | **OnPrem** k3s: NFS MongoDB + in-cluster **Ollama** |
| [`deployments/k8s/k3s/`](https://github.com/BuildAndDestroy/ReaperC2/tree/main/deployments/k8s/k3s/) | Short pointer: k3s + DocumentDB → `reaperc2/` |

Legacy samples: [`full-deployment.yaml`](https://github.com/BuildAndDestroy/ReaperC2/blob/main/deployments/k8s/full-deployment.yaml), [`OnPrem/full-deployment.yaml`](https://github.com/BuildAndDestroy/ReaperC2/blob/main/deployments/k8s/OnPrem/full-deployment.yaml).

Always review placeholders (registry secrets, Mongo credentials, hostnames, TLS) before applying.

## Build and push the image

**DocumentDB (EKS / k3s):**

```bash
cd deployments/k8s/reaperc2
./build-push-image.sh --arch amd64    # or arm64 for Pi nodes
```

**OnPrem NFS + Ollama:**

```bash
cp deployments/k3s/config.example.env deployments/k3s/config.env
./deployments/k3s/scripts/build-push-image.sh
```

Requires **Go** on the build machine for the OnPrem script (host cross-compile + `Dockerfile.pack`).

## Two listeners: beacon vs admin

The binary listens on **8080** (beacon) and **8443** (admin) by default.

- **Expose 8080** through Ingress / LoadBalancer for implant traffic. Configure `BEACON_PUBLIC_BASE_URL` (and per-beacon base URL in the UI) to that **public** HTTPS origin.
- **Do not** publish **8443** on a public Ingress for routine operation. Use **`kubectl port-forward`** (or an SSH tunnel via a bastion) from a trusted workstation to reach the admin UI at `http://127.0.0.1:8443` on that machine.

```bash
kubectl port-forward -n reaperc2-ns deployment/reaperc2-deployment 8443:8443
```

## Apply manifests

**DocumentDB:**

```bash
cd deployments/k8s/reaperc2
REAPER_CLUSTER=k3s ./deploy-cluster.sh all    # or default aws for EKS
./deploy-cluster.sh apply-ingress
```

**OnPrem NFS + Ollama:**

```bash
./deployments/k3s/scripts/deploy.sh
```

## MongoDB vs DocumentDB

- **OnPrem / in-cluster Mongo**: `deployments/k3s/scripts/deploy.sh` or [`test/setup_mongo.sh`](https://github.com/BuildAndDestroy/ReaperC2/blob/main/test/setup_mongo.sh).
- **DocumentDB**: [`deployments/k8s/reaperc2/README.md`](https://github.com/BuildAndDestroy/ReaperC2/tree/main/deployments/k8s/reaperc2/README.md).

## Operator AI (multi-model)

- [`deployments/k8s/operator-ai.yaml`](https://github.com/BuildAndDestroy/ReaperC2/blob/main/deployments/k8s/operator-ai.yaml) — template; copy to `operator-ai.local.yaml` for secrets.
- **In-cluster Ollama (OnPrem):** [`deployments/k8s/ollama.yaml`](https://github.com/BuildAndDestroy/ReaperC2/blob/main/deployments/k8s/ollama.yaml) + enable Ollama keys in `operator-ai.local.yaml`, or `deployments/k3s/scripts/deploy.sh`.

See [Operator AI](/documentation/operator-guide-ai) for variable details.

## See also

- [Installation](/documentation/installation)
- [Docker Compose](/documentation/docker-compose)
- [Usage](/documentation/usage)
