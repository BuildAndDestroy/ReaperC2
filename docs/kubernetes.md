# Kubernetes

Example and environment-specific manifests live under [`deployments/k8s/`](https://github.com/BuildAndDestroy/ReaperC2/tree/main/deployments/k8s/), including:

- [`full-deployment.yaml`](https://github.com/BuildAndDestroy/ReaperC2/blob/main/deployments/k8s/full-deployment.yaml) — namespace, MongoDB, ReaperC2, Traefik **IngressRoute**, beacon-facing service (sample placeholders for secrets and NFS).
- [`OnPrem/`](https://github.com/BuildAndDestroy/ReaperC2/tree/main/deployments/k8s/OnPrem/) and [`AWS/`](https://github.com/BuildAndDestroy/ReaperC2/tree/main/deployments/k8s/AWS/) — variants and Terraform notes.

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

## MongoDB in-cluster

The sample includes a MongoDB Deployment and PVC. For managed Mongo (DocumentDB, Atlas, etc.), remove the in-cluster Mongo pieces and set `MONGO_HOST` / credentials via Secrets to your provider’s connection string pattern. The [`AWS/`](https://github.com/BuildAndDestroy/ReaperC2/tree/main/deployments/k8s/AWS/) tree has DocumentDB-oriented notes.

## Seeding the database

From a pod or job that can reach MongoDB, run the repo’s [`test/setup_mongo.sh`](https://github.com/BuildAndDestroy/ReaperC2/blob/main/test/setup_mongo.sh) with `MONGO_HOST` set to the cluster DNS name of the Mongo Service (see root README “Kubernetes” example).

## See also

- [Installation](/documentation/installation)
- [Docker Compose](/documentation/docker-compose)
- [Usage](/documentation/usage)
