# Kubernetes

Example and environment-specific manifests live under [`deployments/k8s/`](https://github.com/BuildAndDestroy/ReaperC2/tree/main/deployments/k8s/), including:

- [`full-deployment.yaml`](https://github.com/BuildAndDestroy/ReaperC2/blob/main/deployments/k8s/full-deployment.yaml) — namespace, MongoDB, ReaperC2, Traefik **IngressRoute**, beacon-facing service (sample placeholders for secrets and NFS).
- [`OnPrem/`](https://github.com/BuildAndDestroy/ReaperC2/tree/main/deployments/k8s/OnPrem/) — in-cluster MongoDB + ReaperC2.
- [`AWS/`](https://github.com/BuildAndDestroy/ReaperC2/tree/main/deployments/k8s/AWS/) — ReaperC2 on existing EKS + **DocumentDB** + Traefik (`kubectl apply -k deployments/k8s/AWS`); no in-cluster MongoDB or Terraform in this repo.

Always review and replace **placeholders** (registry pull secrets, Mongo credentials, storage class, hostnames, TLS issuers) before applying to a real cluster.

## Build and push the image

```bash
git submodule update --init --recursive   # recommended
docker build -t <your-registry>/reaperc2:<tag> .
docker push <your-registry>/reaperc2:<tag>
```

Point the Deployment image to your pushed tag. If CI cannot init the submodule, pass `--build-arg SCYTHE_GIT_REF=<tag>` so the Dockerfile clone matches the Scythe revision you intend to ship.

## Two listeners: beacon vs admin

The binary listens on **8080** (beacon) and **8443** (admin) by default.

- **Expose 8080** through Ingress / LoadBalancer for implant traffic. Configure `BEACON_PUBLIC_BASE_URL` (and per-beacon base URL in the UI) to that **public** HTTPS origin.
- **Do not** publish **8443** on a public Ingress for routine operation. Use **`kubectl port-forward`** (or an SSH tunnel via a bastion) from a trusted workstation to reach the admin UI at `http://127.0.0.1:8443` on that machine.

Example:

```bash
kubectl port-forward -n reaperc2-ns deployment/reaperc2-deployment 8443:8443
```

Adjust namespace and resource names to match your manifests.

## Apply manifests

After editing YAML for your cluster:

```bash
kubectl apply -f deployments/k8s/full-deployment.yaml
```

Traefik **IngressRoute** in the sample routes **beacon** traffic. If you use another ingress controller, adapt routes and TLS the same way: **only** the beacon service port should be on the public edge unless you deliberately expose the admin UI.

## MongoDB vs DocumentDB

- **Root / OnPrem** [`full-deployment.yaml`](https://github.com/BuildAndDestroy/ReaperC2/blob/main/deployments/k8s/full-deployment.yaml) includes an in-cluster MongoDB Deployment and PVC.
- **AWS** uses **Amazon DocumentDB** only: apply [`deployments/k8s/AWS/examples/documentdb-secret.yaml`](https://github.com/BuildAndDestroy/ReaperC2/tree/main/deployments/k8s/AWS/examples/documentdb-secret.yaml), fetch the RDS CA bundle (`fetch-docdb-ca-bundle.sh`), then `kubectl apply -k deployments/k8s/AWS`. With `DEPLOY_ENV=AWS`, the app adds DocumentDB TLS query parameters automatically ([`pkg/dbconnections/mongoconnections.go`](https://github.com/BuildAndDestroy/ReaperC2/blob/main/pkg/dbconnections/mongoconnections.go)).

## Seeding the database

- **OnPrem / in-cluster Mongo**: [`test/setup_mongo.sh`](https://github.com/BuildAndDestroy/ReaperC2/blob/main/test/setup_mongo.sh) with `MONGO_HOST` set to the Mongo Service DNS name.
- **AWS DocumentDB**: run `docdb-init-job.yaml` and `docdb-init-user-job.yaml` from [`deployments/k8s/AWS/`](https://github.com/BuildAndDestroy/ReaperC2/tree/main/deployments/k8s/AWS); use your infra repo or a host with the RDS CA bundle (`fetch-docdb-ca-bundle.sh`) for ad-hoc `mongosh`.

## Operator AI (multi-model)

The same env vars as Docker Compose / `.env.example` apply in Kubernetes. Sample manifests:

- [`deployments/k8s/operator-ai.yaml`](https://github.com/BuildAndDestroy/ReaperC2/blob/main/deployments/k8s/operator-ai.yaml) — **ConfigMap** (model catalog, defaults) and **Secret** (API keys).
- ReaperC2 Deployments under `deployments/k8s/**/full-deployment.yaml` load them with `envFrom` (`optional: true` so the app starts before you enable AI).

**Enable Operator AI:**

1. Edit `operator-ai.yaml`: set `REAPER_AI_OPENAI_MODELS` / `REAPER_AI_ANTHROPIC_MODELS` (comma-separated) and API keys in the Secret.
2. Apply:

   ```bash
   kubectl apply -f deployments/k8s/operator-ai.yaml
   kubectl rollout restart deployment/reaperc2-deployment -n reaperc2-ns
   ```

3. Port-forward admin (`8443`) and open **Operator AI** — choose **Auto** or a specific model from the dropdown.

| Source | Variables |
|--------|-----------|
| ConfigMap `reaperc2-ai-config` | `REAPER_AI_ENABLED`, `REAPER_AI_DEFAULT_MODEL`, `REAPER_AI_*_MODELS`, `REAPER_AI_MODELS`, `REAPER_AI_MAX_TOKENS` |
| Secret `reaperc2-ai-secrets` | `REAPER_AI_OPENAI_API_KEY`, `REAPER_AI_ANTHROPIC_API_KEY`, optional `REAPER_AI_BEDROCK_*` keys (or Bedrock via IRSA — see operator guide) |

**Ollama in K8s** is optional. Most deployments use cloud APIs only. If you run Ollama as its own Deployment/Service, set `REAPER_AI_OLLAMA_ENABLED=1` and `REAPER_AI_OLLAMA_API_URL` to the in-cluster URL (for example `http://ollama.ollama-ns.svc.cluster.local:11434/v1`) in the ConfigMap — not `host.docker.internal`.

To patch keys without storing them in git:

```bash
kubectl create secret generic reaperc2-ai-secrets -n reaperc2-ns \
  --from-literal=REAPER_AI_OPENAI_API_KEY=sk-... \
  --from-literal=REAPER_AI_ANTHROPIC_API_KEY=sk-ant-... \
  --dry-run=client -o yaml | kubectl apply -f -
```

See [Operator AI](/documentation/operator-guide-ai) for variable details.

## See also

- [Installation](/documentation/installation)
- [Docker Compose](/documentation/docker-compose)
- [Usage](/documentation/usage)
