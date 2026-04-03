# DigitalOcean: Kubernetes and MongoDB (Terraform)

This directory provisions **supporting infrastructure** for ReaperC2 on DigitalOcean: a VPC, a DigitalOcean Kubernetes (DOKS) cluster, and a **managed MongoDB** instance attached to the same VPC so the cluster can reach the database over the private network.

Application manifests (Deployments, Services, Ingress, etc.) live under `deployments/k8s/` elsewhere; use those as a template after the cluster exists.

## What Terraform creates

| Resource | Details |
|----------|---------|
| VPC | `do-k8s-vpc` in `var.region` |
| Kubernetes | Cluster `my-k8s-cluster`, Kubernetes version `latest`, 2× `s-1vcpu-2gb` worker nodes |
| MongoDB | Managed cluster `mongo-cluster`, MongoDB 6, `db-s-1vcpu-1gb`, 1 node, 15 GiB storage, same VPC |

## Prerequisites

- [Terraform](https://developer.hashicorp.com/terraform/install) ≥ 1.0
- A DigitalOcean account and [API token](https://docs.digitalocean.com/reference/api/create-personal-access-token/) with read/write access
- Optional: `kubectl` if you will deploy workloads immediately after

## Configure and apply

From this directory:

```bash
export DIGITALOCEAN_TOKEN="your_personal_access_token"
# Optional: export TF_VAR_do_token="$DIGITALOCEAN_TOKEN" instead of passing -var below

terraform init

terraform plan -out=tfplan -var "do_token=$DIGITALOCEAN_TOKEN"

terraform apply tfplan
```

Or apply in one step without a saved plan:

```bash
terraform apply -var "do_token=$DIGITALOCEAN_TOKEN"
```

### Region

Default region is `nyc3`. Override in `variables.tf` or on the CLI:

```bash
terraform plan -out=tfplan \
  -var "do_token=$DIGITALOCEAN_TOKEN" \
  -var "region=lon1"
terraform apply tfplan
```

## Outputs

After apply, Terraform exposes:

- **`kubeconfig`**: raw DOKS kubeconfig (marked sensitive).
- **`mongodb_private_uri`**: private connection URI for the managed MongoDB (sensitive).

View without printing secrets to the screen unnecessarily:

```bash
terraform output
```

Write kubeconfig to a **dedicated file** (recommended instead of overwriting `~/.kube/config`):

```bash
terraform output -raw kubeconfig > ./kubeconfig-do.yaml
export KUBECONFIG="$(pwd)/kubeconfig-do.yaml"
kubectl get nodes
```

If you already use another cluster, either swap `KUBECONFIG` per shell or merge contexts with `kubectl config` — avoid blindly redirecting to `~/.kube/config` unless that file is disposable.

Fetch the Mongo URI when scripting:

```bash
terraform output -raw mongodb_private_uri
```

## Deploying ReaperC2 on this cluster

1. **Build and push** a container image to a registry your DOKS nodes can pull (DigitalOcean Container Registry, Docker Hub, etc.) and create an `imagePullSecret` if the registry is private.

2. **Environment**: ReaperC2’s valid `DEPLOY_ENV` values today are `AWS`, `AZURE`, `GCP`, and `ONPREM` (`pkg/deploymehere/deploymehere.go`). For DigitalOcean, use **`ONPREM`** unless you add a dedicated value in code. That path uses the standard MongoDB URI builder (not AWS DocumentDB TLS defaults).

3. **MongoDB variables**: Map your managed DB credentials into the same env vars used in other manifests (`MONGO_HOST`, `MONGO_PORT`, `MONGO_USERNAME`, `MONGO_PASSWORD`, `MONGO_DATABASE`). Derive them from the DigitalOcean control panel or from `mongodb_private_uri`.  
   DigitalOcean managed databases commonly require TLS; if connections fail, set **`MONGO_USE_TLS=true`** (see `pkg/dbconnections/mongoconnections.go`).

4. **Manifests**: Start from `deployments/k8s/OnPrem/full-deployment.yaml` or `deployments/k8s/AWS/full-deployment.yaml` and replace registry URLs, secrets, and resource sizes for your image and DO networking.

Because the database is on the **VPC private network**, use the **private** host/URI from DigitalOcean for traffic from pods inside this cluster.

## Tear down

```bash
terraform destroy -var "do_token=$DIGITALOCEAN_TOKEN"
```

Confirm you are destroying the intended resources; this removes the cluster, database, and VPC created by this configuration.

## Files

- `main.tf` — VPC, DOKS, managed MongoDB, outputs  
- `variables.tf` — `do_token`, `region`  

For DigitalOcean-specific product docs, see [Kubernetes](https://docs.digitalocean.com/products/kubernetes/) and [Managed Databases](https://docs.digitalocean.com/products/databases/).
