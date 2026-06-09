# AWS path (legacy)

ReaperC2 Kubernetes manifests and automation now live in **[`../reaperc2/`](../reaperc2/README.md)** (shared **DocumentDB + Traefik + cert-manager** flow for **EKS and k3s**).

- **Quick path (unchanged):** `kubectl apply -k .` from this directory still applies the **aws-ecr** overlay (same as before).
- **Scripts:** run [`../reaperc2/deploy-cluster.sh`](../reaperc2/deploy-cluster.sh), or keep using `./deploy-aws-k8s.sh` here — it delegates to that script with `REAPER_CLUSTER=aws`.

**k3s:** `cd ../reaperc2 && REAPER_CLUSTER=k3s ./deploy-cluster.sh help` — see the main README for the k3s overlay and ECR opt-in.
