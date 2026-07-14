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
| `MONGO_NFS_SERVER` / `MONGO_NFS_PATH` | NFS export for MongoDB PV |
| `REGISTRY_SERVER` / `REGISTRY_USERNAME` / `REGISTRY_PASSWORD` | Private registry push/pull |
| `REGISTRY_SECRET_NAME` | Kubernetes pull secret name (default `registry-credentials`) |
| `REAPERC2_IMAGE` / `REAPERC2_IMAGE_TAG` | Container image to deploy |
| `OLLAMA_PULL_MODELS` | Comma-separated tags pulled on first Ollama start (e.g. `llama3.2:latest,gpt-oss:latest`) |
| `REAPERC2_NAMESPACE` | Default `reaperc2-ns` |
| `KUBECONFIG` | Optional kubeconfig path |

**DocumentDB / ECR builds** use [`k8s/reaperc2/build-push-image.sh`](k8s/reaperc2/build-push-image.sh) and the Makefile (`AWS_ACCOUNT_ID`, `AWS_REGION`, etc.) — not `config.env`.

---

## Mongo secrets (OnPrem `k3s/`)

Edit [`k3s/mongo-secret.yaml`](k3s/mongo-secret.yaml) before first deploy (`root_*`, `app_*`). Do not change passwords after Mongo has initialized the NFS volume without resetting data.

---

## Operator AI

[`k8s/operator-ai.yaml`](k8s/operator-ai.yaml) — ConfigMap + Secret template. Copy to `operator-ai.local.yaml` (gitignored) for real keys.

| Concern | Variable / file |
|---------|-----------------|
| Dropdown models (comma-separated per provider) | `REAPER_AI_OPENAI_MODELS`, `REAPER_AI_ANTHROPIC_MODELS`, `REAPER_AI_OLLAMA_MODELS`, … |
| In-cluster Ollama | Apply [`k8s/ollama.yaml`](k8s/ollama.yaml); set `REAPER_AI_OLLAMA_API_URL=http://ollama.ollama-ns.svc.cluster.local:11434/v1` |
| Pull models into Ollama | `OLLAMA_PULL_MODELS` in `k3s/config.env` |

Full variable list: [docs/operator-guide-ai.md](../docs/operator-guide-ai.md).

---

## See also

- [docs/kubernetes.md](../docs/kubernetes.md)
- [k8s/reaperc2/README.md](k8s/reaperc2/README.md)
