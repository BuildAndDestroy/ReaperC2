# Usage

```
export DIGITALOCEAN_TOKEN="your_do_token"

terraform init

terraform plan -out=tfplan

terraform apply -var "do_token=$DIGITALOCEAN_TOKEN"

terraform output kubeconfig > ~/.kube/config
```
