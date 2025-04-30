# Usage

## Prerequisites

* eksctl
* terraform
* IAM user with Policies privileges to EKS, EC2, Cloudformation, ELB, IAM, RDS, and DocDB
* SSH key in AWS
* Update usernames and passwords in the mongoconnections.go file. This will be a config file later
* Update full_deployment.yaml with correct FQDN
* Update terraform files with new usernames and passwords


## Setting up your AWS profile and running ekscli

* Set your profile if not default

```
export AWS_PROFILE=your-new-iam-profile
aws sts get-caller-identity
```

## EKS Deploy

* Run eksctl dry runs and deploy. Replace key file

```
eksctl create cluster --name=test-cluster-1 --region=us-east-1 --without-nodegroup --dry-run > 01-create-cluster.yaml

eksctl create cluster -f 01-create-cluster.yaml

eksctl create nodegroup --cluster=test-cluster-1 --nodes 2 --nodes-min 2 --nodes-max 5 --asg-access --region us-east-1 --ssh-access --ssh-public-key /home/ubuntu/.ssh/eks-keys.pub --instance-types=t3.large --node-volume-size=30 --spot --max-pods-per-node 100 --dry-run > 02-x86-64-create-nodegroup.yaml

eksctl create nodegroup -f 02-x86-64-create-nodegroup.yaml
```

* Update your kube config
```
aws eks update-kubeconfig --region us-east-1 --name test-cluster-1
```

## DocumentDB Deploy

* This wil deploy DocumentDB that is only accessable from the EKS cluster network

```

terraform init

terraform plan -out="tfplan"

terraform apply

```

## Note

* If you change the name of your cluster, which you should, make sure to update main.tf with your cluster name. Otherwise terraform will fail.

```
aws ec2 describe-vpcs --region us-east-1 --query "Vpcs[*].{VpcId:VpcId,Tags:Tags}" --output table

sed -i 's/test-cluster-1/new-cluster-name/g' main.tf
```

* Get Bundle For Encrypted Connections
```
curl -o rds-combined-ca-bundle.pem https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem

cat rds-combined-ca-bundle.pem
```

* Copy the output and put this in the full-deployment.yaml, at the bottom
* Obtain connection strings for the applications

```
terraform output docdb_connection_string

mongosh 'mongodb://username:password@prodsec-docdb-cluster.cluster-XXXXXXXXX.us-east-1.docdb.amazonaws.com:27017/?tls=true&tlsCAFile=/certs/rds-combined-ca-bundle.pem&replicaSet=rs0&readPreference=secondaryPreferred&retryWrites=false'
```

## Init the DB
```
cat setup_documentdb.sh | base64 -w 0

kubectl exec -it docdb-tester -- bash
```

# Destroy Cluster

```
eksctl delete nodegroup -f 02-x86-64-create-nodegroup.yaml --approve

eksctl delete cluster -f 01-create-cluster.yaml
```