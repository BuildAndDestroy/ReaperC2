# k3s / OnPrem NFS + Ollama

For **OnPrem** k3s (NFS MongoDB, in-cluster Ollama, private registry), use **[`../../k3s/`](../../k3s/)** — not this file.

This entry is for **k3s + Amazon DocumentDB** (same as EKS):

```bash
cd ../reaperc2
export REAPER_CLUSTER=k3s
./deploy.sh help          # or: ./deploy-cluster.sh help
./deploy.sh all
./deploy.sh apply-ingress
```

Core apply is `kubectl apply -k ../reaperc2/overlays/k3s` (no ECR `imagePullSecrets` patch). Use `REAPER_ECR_SECRET=1 ./deploy-cluster.sh ecr-secret` if you still pull from ECR.

Build an **arm64** image for ARM nodes: from `../reaperc2`, run `./build-push-image.sh --arch arm64` (or `make build-arm64` from the repo root), then point `base/deployment.yaml` at that tag.
