data "aws_vpc" "selected" {
  id = var.vpc_id
}

data "aws_subnet" "selected" {
  id = var.subnet_id
}

data "aws_route_tables" "subnet_associated" {
  vpc_id = var.vpc_id

  filter {
    name   = "association.subnet-id"
    values = [var.subnet_id]
  }
}

data "aws_route_tables" "main" {
  vpc_id = var.vpc_id

  filter {
    name   = "association.main"
    values = ["true"]
  }
}

locals {
  selected_route_table_id = length(data.aws_route_tables.subnet_associated.ids) > 0 ? one(data.aws_route_tables.subnet_associated.ids) : one(data.aws_route_tables.main.ids)
}

data "aws_route_table" "selected" {
  route_table_id = local.selected_route_table_id
}

data "aws_ami" "al2023_arm64" {
  count       = var.ami_id == null ? 1 : 0
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["al2023-ami-2023.*-kernel-6.1-arm64"]
  }

  filter {
    name   = "architecture"
    values = ["arm64"]
  }

  filter {
    name   = "root-device-type"
    values = ["ebs"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

locals {
  ami_id = coalesce(var.ami_id, try(data.aws_ami.al2023_arm64[0].id, null))
  tags = merge(var.tags, {
    Project   = "expense-tracker"
    Component = "serverless-postgres"
    ManagedBy = "terraform"
  })
}

resource "aws_security_group" "postgres" {
  name_prefix = "${var.name_prefix}-host-"
  description = "Minimal PostgreSQL EC2 host"
  vpc_id      = var.vpc_id
  tags        = merge(local.tags, { Name = "${var.name_prefix}-host" })

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_security_group" "worker_client" {
  name_prefix = "${var.name_prefix}-worker-"
  description = "Future worker Lambda PostgreSQL client"
  vpc_id      = var.vpc_id
  tags        = merge(local.tags, { Name = "${var.name_prefix}-worker" })

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_security_group" "bootstrap_client" {
  name_prefix = "${var.name_prefix}-bootstrap-"
  description = "Future bootstrap Lambda PostgreSQL client"
  vpc_id      = var.vpc_id
  tags        = merge(local.tags, { Name = "${var.name_prefix}-bootstrap" })

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_vpc_security_group_ingress_rule" "ssh" {
  count = var.enable_temporary_public_access ? 1 : 0

  security_group_id = aws_security_group.postgres.id
  cidr_ipv4         = var.operator_ssh_cidr
  ip_protocol       = "tcp"
  from_port         = 22
  to_port           = 22
  description       = "Temporary operator SSH"
}

resource "aws_vpc_security_group_ingress_rule" "postgres_from_worker" {
  security_group_id            = aws_security_group.postgres.id
  referenced_security_group_id = aws_security_group.worker_client.id
  ip_protocol                  = "tcp"
  from_port                    = 5432
  to_port                      = 5432
  description                  = "Worker Lambda PostgreSQL access"
}

resource "aws_vpc_security_group_ingress_rule" "postgres_from_bootstrap" {
  security_group_id            = aws_security_group.postgres.id
  referenced_security_group_id = aws_security_group.bootstrap_client.id
  ip_protocol                  = "tcp"
  from_port                    = 5432
  to_port                      = 5432
  description                  = "Bootstrap Lambda PostgreSQL access"
}

resource "aws_vpc_security_group_egress_rule" "host_outbound" {
  security_group_id = aws_security_group.postgres.id
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "-1"
  description       = "Package installation and normal outbound access"
}

resource "aws_vpc_security_group_egress_rule" "worker_to_postgres" {
  security_group_id            = aws_security_group.worker_client.id
  referenced_security_group_id = aws_security_group.postgres.id
  ip_protocol                  = "tcp"
  from_port                    = 5432
  to_port                      = 5432
  description                  = "Worker to PostgreSQL"
}

resource "aws_vpc_security_group_egress_rule" "bootstrap_to_postgres" {
  security_group_id            = aws_security_group.bootstrap_client.id
  referenced_security_group_id = aws_security_group.postgres.id
  ip_protocol                  = "tcp"
  from_port                    = 5432
  to_port                      = 5432
  description                  = "Bootstrap to PostgreSQL"
}

resource "aws_instance" "postgres" {
  ami                         = local.ami_id
  instance_type               = var.instance_type
  subnet_id                   = var.subnet_id
  key_name                    = var.key_pair_name
  associate_public_ip_address = false
  vpc_security_group_ids      = [aws_security_group.postgres.id]

  metadata_options {
    http_endpoint = "enabled"
    http_tokens   = "required"
  }

  root_block_device {
    encrypted             = true
    volume_type           = "gp3"
    volume_size           = 10
    delete_on_termination = true
  }

  tags = merge(local.tags, { Name = var.name_prefix })

  lifecycle {
    precondition {
      condition     = data.aws_subnet.selected.vpc_id == var.vpc_id
      error_message = "subnet_id must belong to vpc_id."
    }

    precondition {
      condition = anytrue([
        for route in data.aws_route_table.selected.routes :
        try(route.cidr_block, "") == "0.0.0.0/0" && startswith(try(route.gateway_id, ""), "igw-")
      ])
      error_message = "subnet_id must have an effective 0.0.0.0/0 route to an Internet Gateway for temporary EIP access."
    }
  }
}

resource "aws_eip" "temporary_postgres" {
  count = var.enable_temporary_public_access ? 1 : 0

  domain = "vpc"
  tags   = merge(local.tags, { Name = "${var.name_prefix}-temporary-access" })

  lifecycle {
    precondition {
      condition     = var.operator_ssh_cidr != null
      error_message = "operator_ssh_cidr is required when temporary public access is enabled."
    }
  }
}

resource "aws_eip_association" "temporary_postgres" {
  count = var.enable_temporary_public_access ? 1 : 0

  instance_id   = aws_instance.postgres.id
  allocation_id = aws_eip.temporary_postgres[0].id
}
