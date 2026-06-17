# Deploy ReaperC2 on Kubernetes (quick pointer)

The supported path is **`deployments/k8s/reaperc2/`** with **DocumentDB**, Traefik, and cert-manager.

## Read this first

- **[reaperc2/README.md](reaperc2/README.md)** — full checklist, DocumentDB pitfalls, ingress order, troubleshooting.
- **[reaperc2/examples/README.md](reaperc2/examples/README.md)** — copying secret templates to `*.local.yaml`.

## Shortest path

```bash
cd deployments/k8s/reaperc2
chmod +x deploy.sh reroll.sh build-push-image.sh deploy-cluster.sh base/fetch-docdb-ca-bundle.sh
# Build and push ECR image (pick arch): ./build-push-image.sh --arch amd64   # or arm64 / both
# Copy and edit examples/*.local.yaml, base/deployment.yaml image, optional ../operator-ai.local.yaml
./deploy.sh check-local
./deploy.sh all                    # or: ./deploy-cluster.sh all
./deploy.sh job-docdb-user
./deploy.sh job-docdb-init         # optional
./deploy.sh apply-ingress          # when Traefik + cert-manager are ready
```

**Egress lockdown (optional):** copy `reaperc2/examples/networkpolicy-egress-restricted.yaml` → `networkpolicy-egress-restricted.local.yaml`, edit DocumentDB CIDR, then `./deploy.sh --with-egress all`. Requires a CNI that enforces NetworkPolicy.

**After you change secrets or the image tag:** `./reroll.sh` or `./reroll.sh --apply-secrets` / `--refresh-ecr`.

**k3s:** `REAPER_CLUSTER=k3s ./deploy.sh all` (same scripts).

Legacy / other layouts: [docs/kubernetes.md](../docs/kubernetes.md).
