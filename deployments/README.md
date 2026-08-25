# Deployment layouts

Two supported Kubernetes paths — pick one per cluster:

| Path | Use when | Entry |
|------|----------|--------|
| **[`k8s/reaperc2/`](k8s/reaperc2/)** | **DocumentDB** on EKS or k3s (Traefik, cert-manager, Bedrock Operator AI) | [`k8s/DEPLOY.md`](k8s/DEPLOY.md) |
| **[`k3s/`](k3s/)** | **OnPrem** homelab / Pi: NFS MongoDB, in-cluster **Ollama**, private registry | [`k3s/README.md`](k3s/README.md) |

**k3s + DocumentDB** uses the **reaperc2** path (`REAPER_CLUSTER=k3s`), not `deployments/k3s/`. See [`k8s/k3s/README.md`](k8s/k3s/README.md).

Legacy samples: [`k8s/full-deployment.yaml`](k8s/full-deployment.yaml), [`k8s/OnPrem/full-deployment.yaml`](k8s/OnPrem/full-deployment.yaml).

---

## `config.env` (OnPrem `deployments/k3s/` only)

```bash
cp deployments/k3s/config.example.env deployments/k3s/config.env
```

Or one shared file: `deployments/config.env` (scripts check `k3s/config.env` first, then `deployments/config.env`).

**Special characters:** quote passwords with `#` or `!`:

```env
REGISTRY_PASSWORD='your-password-here'
```

| Variable | Purpose |
|----------|---------|
| `MONGO_NFS_SERVER` / `MONGO_NFS_PATH` | NFS export for MongoDB PV (e.g. `192.168.1.100:/export/reaperc2-metric`) |
| `MONGO_DATABASE` | Mongo database name (default `reaperc2-metric`) |
| `MONGO_NODE_HOSTNAME` | Node for Mongo StatefulSet (e.g. `control-plane`; mongo:7 may need ARMv8.2+) |
| `REGISTRY_SERVER` / `REGISTRY_USERNAME` / `REGISTRY_PASSWORD` | Private registry push/pull |
| `REGISTRY_SECRET_NAME` | Kubernetes pull secret name (default `registry-credentials`) |
| `REAPERC2_IMAGE` / `REAPERC2_IMAGE_TAG` | Container image to deploy |
| `INGRESS_HOST` | Public beacon hostname (default `beacons.example.com`) |
| `CERT_MANAGER_ISSUER` | ClusterIssuer for beacon TLS (default `letsencrypt-prod`) |
| `OLLAMA_NFS_SERVER` / `OLLAMA_NFS_PATH` | NFS export for Ollama model weights (e.g. `/export/ollama-models`) |
| `OLLAMA_PULL_MODELS` | Comma-separated tags pulled on first Ollama start |
| `OLLAMA_STORAGE_CLASS` | Fallback PVC class only if `OLLAMA_NFS_*` is unset |
| `REAPERC2_NAMESPACE` | Default `reaperc2-ns` |
| `KUBECONFIG` | Optional kubeconfig path |

**DocumentDB / ECR builds** use [`k8s/reaperc2/build-push-image.sh`](k8s/reaperc2/build-push-image.sh) and the Makefile (`AWS_ACCOUNT_ID`, `AWS_REGION`, etc.) — not `config.env`.

---

## Mongo secrets (OnPrem `k3s/`)

Edit [`k3s/mongo-secret.yaml`](k3s/mongo-secret.yaml) before first deploy (`root_*`, `app_*`). Database name is **`reaperc2-metric`** (`MONGO_DATABASE`). Do not change passwords after Mongo has initialized the NFS volume without resetting data.

**Ingress:** beacons only on `INGRESS_HOST` (default `beacons.example.com`). Admin UI uses `kubectl port-forward … 8443:8443` — not exposed on ingress (same as [`k8s/reaperc2/`](k8s/reaperc2/)).

---

## Operator AI

[`k8s/operator-ai.yaml`](k8s/operator-ai.yaml) — ConfigMap + Secret template. Copy to `operator-ai.local.yaml` (gitignored) for real keys.

| Concern | Variable / file |
|---------|-----------------|
| Dropdown models (comma-separated per provider) | `REAPER_AI_OPENAI_MODELS`, `REAPER_AI_ANTHROPIC_MODELS`, `REAPER_AI_OLLAMA_MODELS`, … |
| In-cluster Ollama (default) | Applied via `k8s/ollama.yaml`; `REAPER_AI_OLLAMA_*` enabled in `operator-ai.yaml` |
| Pull models into Ollama | `OLLAMA_PULL_MODELS` in `ollama.yaml` ConfigMap (or `k3s/config.env`) |

Full variable list: [docs/operator-guide-ai.md](../docs/operator-guide-ai.md).

---

## See also

- [docs/kubernetes.md](../docs/kubernetes.md)
- [k8s/reaperc2/README.md](k8s/reaperc2/README.md)
