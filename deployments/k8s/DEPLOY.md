# Deploy ReaperC2 on Kubernetes (quick pointer)

## Choose a layout

| Layout | Directory |
|--------|-----------|
| **DocumentDB** (EKS or k3s) | [`reaperc2/`](reaperc2/) |
| **OnPrem** NFS Mongo + Ollama (k3s homelab / Pi) | [`../k3s/`](../k3s/) |

See [`../README.md`](../README.md) for the full comparison.

---

## DocumentDB (`reaperc2/`)

Supported path: **DocumentDB**, Traefik, cert-manager, Bedrock Operator AI.

## Read this first (DocumentDB)

- **[reaperc2/README.md](reaperc2/README.md)** — full checklist, DocumentDB pitfalls, ingress order, troubleshooting.
- **[reaperc2/examples/README.md](reaperc2/examples/README.md)** — copying secret templates to `*.local.yaml`.

## Shortest path

```bash
cd deployments/k8s/reaperc2
chmod +x deploy.sh reroll.sh ship.sh build-push-image.sh deploy-cluster.sh base/fetch-docdb-ca-bundle.sh
# AWS: export AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_SESSION_TOKEN; unset AWS_PROFILE
# Build/push :latest and restart: ./ship.sh
# First install only: copy examples/*.local.yaml, optional ../operator-ai.local.yaml
./deploy.sh check-local
./deploy.sh all                    # or: ./deploy-cluster.sh all
./deploy.sh job-docdb-user
./deploy.sh job-docdb-init         # optional
./deploy.sh apply-ingress          # when Traefik + cert-manager are ready
```

**Egress lockdown (optional):** copy `reaperc2/examples/networkpolicy-egress-restricted.yaml` → `networkpolicy-egress-restricted.local.yaml`, edit DocumentDB CIDR, then `./deploy.sh --with-egress all`. Requires a CNI that enforces NetworkPolicy.

**After you change secrets or manifests:** `./reroll.sh --apply-core`. **After a new image:** `./ship.sh` (push `:latest` + restart). `./reroll.sh` alone restarts pods without pushing.

**k3s:** `REAPER_CLUSTER=k3s ./deploy.sh all` (same scripts).

Legacy / other layouts: [docs/kubernetes.md](../docs/kubernetes.md).
