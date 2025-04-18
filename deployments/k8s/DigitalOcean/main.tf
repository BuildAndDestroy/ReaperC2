terraform {
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
  }
}

provider "digitalocean" {
  token = var.do_token
}

resource "digitalocean_vpc" "main" {
  name   = "do-k8s-vpc"
  region = var.region
}

resource "digitalocean_kubernetes_cluster" "k8s" {
  name     = "my-k8s-cluster"
  region   = var.region
  version  = "latest"
  vpc_uuid = digitalocean_vpc.main.id

  node_pool {
    name       = "default-pool"
    size       = "s-1vcpu-2gb"
    node_count = 2
    tags       = ["k8s"]
  }
}

resource "digitalocean_database_cluster" "mongodb" {
  name       = "mongo-cluster"
  engine     = "mongodb"
  version    = "6"
  size       = "db-s-1vcpu-1gb"
  region     = var.region
  node_count = 1
  storage_size_mib = 15360
  private_network_uuid = digitalocean_vpc.main.id
}

output "kubeconfig" {
  value     = digitalocean_kubernetes_cluster.k8s.kube_config[0].raw_config
  sensitive = true
}

output "mongodb_private_uri" {
  value     = digitalocean_database_cluster.mongodb.private_uri
  sensitive = true
}
