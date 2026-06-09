# k3s

ReaperC2 on **k3s** uses the same manifests and Traefik/cert-manager flow as EKS — see **[`../reaperc2/README.md`](../reaperc2/README.md)** (section **k3s notes**).

```bash
cd ../reaperc2
export REAPER_CLUSTER=k3s
./deploy-cluster.sh help
./deploy-cluster.sh all
./deploy-cluster.sh apply-ingress
```

Core apply is `kubectl apply -k ../reaperc2/overlays/k3s` (no ECR `imagePullSecrets` patch). Use `REAPER_ECR_SECRET=1 ./deploy-cluster.sh ecr-secret` if you still pull from ECR.
