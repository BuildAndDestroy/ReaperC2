# Deploy ReaperC2 on Kubernetes (quick pointer)

The supported path is **`deployments/k8s/reaperc2/`** with **DocumentDB**, Traefik, and cert-manager.

## Read this first

- **[reaperc2/README.md](reaperc2/README.md)** — full checklist, DocumentDB pitfalls, ingress order, troubleshooting.
- **[reaperc2/examples/README.md](reaperc2/examples/README.md)** — copying secret templates to `*.local.yaml`.

## Shortest path

```bash
cd deployments/k8s/reaperc2
chmod +x deploy.sh reroll.sh build-push-image.sh deploy-cluster.sh base/fetch-docdb-ca-bundle.sh
# AWS: export AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_SESSION_TOKEN; unset AWS_PROFILE
# export AWS_ACCOUNT_ID=… AWS_REGION=us-east-1  (match base/deployment.yaml ECR host)
# Build and push: ./build-push-image.sh --arch amd64   # or arm64 / both
# Copy and edit examples/*.local.yaml, base/deployment.yaml image, optional ../operator-ai.local.yaml
./deploy.sh check-local
./deploy.sh all                    # or: ./deploy-cluster.sh all
./deploy.sh job-docdb-user
./deploy.sh job-docdb-init         # optional
./deploy.sh apply-ingress          # when Traefik + cert-manager are ready
```

**Egress lockdown (optional):** copy `reaperc2/examples/networkpolicy-egress-restricted.yaml` → `networkpolicy-egress-restricted.local.yaml`, edit DocumentDB CIDR, then `./deploy.sh --with-egress all`. Requires a CNI that enforces NetworkPolicy.

**After you change secrets, manifests, or the image tag:** `./reroll.sh --apply-core` if `base/deployment.yaml` (or overlay) changed in git; `./reroll.sh` alone only restarts pods without applying YAML from disk. Add `--apply-secrets` / `--refresh-ecr` as needed.

**k3s:** `REAPER_CLUSTER=k3s ./deploy.sh all` (same scripts).

Legacy / other layouts: [docs/kubernetes.md](../docs/kubernetes.md).
