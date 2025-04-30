provider "aws" {
  region = "us-east-1"
}

# Find the VPC created by eksctl
data "aws_vpc" "eks_vpc" {
  filter {
    name   = "tag:alpha.eksctl.io/cluster-name"
    values = ["test-cluster-1"]
  }
}

# Get private subnets
data "aws_subnets" "private" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.eks_vpc.id]
  }
  filter {
    name   = "tag:kubernetes.io/role/internal-elb"
    values = ["1"]
  }
  tags = {
    "kubernetes.io/role/internal-elb" = "1"
  }
}

# Get the EKS-managed security group (used by worker nodes)
data "aws_security_groups" "eks_node_sg" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.eks_vpc.id]
  }
  filter {
    name   = "group-name"
    values = ["*test-cluster-1*"]
  }
}

# Security group for DocumentDB that allows inbound from EKS node SG
resource "aws_security_group" "docdb_sg" {
  name        = "docdb-sg"
  description = "Allow Mongo from EKS node SG"
  vpc_id      = data.aws_vpc.eks_vpc.id

  ingress {
    description     = "MongoDB access from EKS"
    from_port       = 27017
    to_port         = 27017
    protocol        = "tcp"
    security_groups = data.aws_security_groups.eks_node_sg.ids
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_docdb_subnet_group" "docdb_subnets" {
  name       = "docdb-subnet-group"
  subnet_ids = data.aws_subnets.private.ids
}

resource "aws_docdb_cluster" "docdb" {
  cluster_identifier      = "test-cluster"
  engine                  = "docdb"
  master_username         = "docdbadmin"
  master_password         = "PuTY0u4Pa$$w04d!H343!"
  backup_retention_period = 1
  preferred_backup_window = "07:00-09:00"
  db_subnet_group_name    = aws_docdb_subnet_group.docdb_subnets.name
  vpc_security_group_ids  = [aws_security_group.docdb_sg.id]
  skip_final_snapshot     = true
}

resource "aws_docdb_cluster_instance" "docdb_instance" {
  count              = 1
  identifier         = "test-docdb-instance-${count.index}"
  cluster_identifier = aws_docdb_cluster.docdb.id
  instance_class     = "db.t3.medium"
}

output "docdb_connection_string" {
  description = "Connection string for the DocumentDB cluster"
  value = "mongodb://${aws_docdb_cluster.docdb.master_username}:${var.db_password}@${aws_docdb_cluster.docdb.endpoint}:27017/?ssl=true&replicaSet=rs0&readPreference=secondaryPreferred&retryWrites=false"
  sensitive = true
}
