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
  count       = var.database_ami_id == null ? 1 : 0
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
  database_ami_id = coalesce(var.database_ami_id, try(data.aws_ami.al2023_arm64[0].id, null))
}

resource "aws_security_group" "postgres" {
  name_prefix = "${local.resource_prefix}-postgres-host-"
  description = "Expense Tracker PostgreSQL host"
  vpc_id      = var.vpc_id
  tags = merge(local.common_tags, {
    Name = "${local.resource_prefix}-postgres-host", Component = "database"
  })
  lifecycle {
    create_before_destroy = true
  }
}
resource "aws_security_group" "worker" {
  name_prefix = "${local.resource_prefix}-worker-"
  description = "Expense Tracker Worker Lambda database client"
  vpc_id      = var.vpc_id
  tags = merge(local.common_tags, {
    Name = "${local.resource_prefix}-worker", Component = "backend"
  })
  lifecycle {
    create_before_destroy = true
  }
}
resource "aws_security_group" "bootstrap" {
  name_prefix = "${local.resource_prefix}-bootstrap-"
  description = "Expense Tracker Bootstrap Lambda database client"
  vpc_id      = var.vpc_id
  tags = merge(local.common_tags, {
    Name = "${local.resource_prefix}-bootstrap", Component = "database"
  })
  lifecycle {
    create_before_destroy = true
  }
}
resource "aws_security_group" "operator_access" {
  name_prefix = "${local.resource_prefix}-postgres-operator-access-"
  description = "EC2 Instance Connect Endpoint for PostgreSQL host operations"
  vpc_id      = var.vpc_id
  tags = merge(local.common_tags, {
    Name = "${local.resource_prefix}-postgres-operator-access", Component = "database-operator-access"
  })
  lifecycle {
    create_before_destroy = true
  }
}
resource "aws_vpc_security_group_egress_rule" "operator_access_to_postgres" {
  security_group_id            = aws_security_group.operator_access.id
  referenced_security_group_id = aws_security_group.postgres.id
  ip_protocol                  = "tcp"
  from_port                    = 22
  to_port                      = 22
  description                  = "EC2 Instance Connect Endpoint to PostgreSQL host SSH"
}
resource "aws_vpc_security_group_ingress_rule" "operator_access_to_postgres" {
  security_group_id            = aws_security_group.postgres.id
  referenced_security_group_id = aws_security_group.operator_access.id
  ip_protocol                  = "tcp"
  from_port                    = 22
  to_port                      = 22
  description                  = "EC2 Instance Connect Endpoint to PostgreSQL host SSH"
}
resource "aws_ec2_instance_connect_endpoint" "operator_access" {
  subnet_id          = var.subnet_id
  security_group_ids = [aws_security_group.operator_access.id]
  preserve_client_ip = false
  tags = merge(local.common_tags, {
    Name = "${local.resource_prefix}-postgres-operator-access", Component = "database-operator-access"
  })
}
resource "aws_vpc_security_group_ingress_rule" "ssh" {
  count             = var.enable_temporary_public_access ? 1 : 0
  security_group_id = aws_security_group.postgres.id
  cidr_ipv4         = var.operator_ssh_cidr
  ip_protocol       = "tcp"
  from_port         = 22
  to_port           = 22
  description       = "Temporary operator SSH"
}
resource "aws_vpc_security_group_ingress_rule" "postgres_from_worker" {
  security_group_id            = aws_security_group.postgres.id
  referenced_security_group_id = aws_security_group.worker.id
  ip_protocol                  = "tcp"
  from_port                    = 5432
  to_port                      = 5432
  description                  = "Worker to PostgreSQL"
}
resource "aws_vpc_security_group_ingress_rule" "postgres_from_bootstrap" {
  security_group_id            = aws_security_group.postgres.id
  referenced_security_group_id = aws_security_group.bootstrap.id
  ip_protocol                  = "tcp"
  from_port                    = 5432
  to_port                      = 5432
  description                  = "Bootstrap to PostgreSQL"
}
resource "aws_vpc_security_group_egress_rule" "postgres_outbound" {
  security_group_id = aws_security_group.postgres.id
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "-1"
  description       = "Package installation and normal outbound access"
}
resource "aws_vpc_security_group_egress_rule" "worker_to_postgres" {
  security_group_id            = aws_security_group.worker.id
  referenced_security_group_id = aws_security_group.postgres.id
  ip_protocol                  = "tcp"
  from_port                    = 5432
  to_port                      = 5432
  description                  = "Worker to PostgreSQL"
}
resource "aws_vpc_security_group_egress_rule" "bootstrap_to_postgres" {
  security_group_id            = aws_security_group.bootstrap.id
  referenced_security_group_id = aws_security_group.postgres.id
  ip_protocol                  = "tcp"
  from_port                    = 5432
  to_port                      = 5432
  description                  = "Bootstrap to PostgreSQL"
}
resource "aws_launch_template" "postgres" {
  name_prefix = "${local.resource_prefix}-postgres-"
  network_interfaces {
    associate_public_ip_address = false
    delete_on_termination       = true
    device_index                = 0
    security_groups             = [aws_security_group.postgres.id]
    subnet_id                   = var.subnet_id

  }
  tags = merge(local.common_tags, {
    Component = "database"
  })
}
resource "aws_instance" "postgres" {
  ami           = local.database_ami_id
  instance_type = var.database_instance_type
  key_name      = var.key_pair_name
  launch_template {
    id      = aws_launch_template.postgres.id
    version = aws_launch_template.postgres.latest_version
  }
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
  tags = merge(local.common_tags, {
    Name = "${local.resource_prefix}-postgres", Component = "database"
  })
  lifecycle {
    precondition {
      condition     = data.aws_subnet.selected.vpc_id == var.vpc_id
      error_message = "subnet_id must belong to vpc_id."
    }
    precondition {
      condition     = anytrue([for route in data.aws_route_table.selected.routes : try(route.cidr_block, "") == "0.0.0.0/0" && startswith(try(route.gateway_id, ""), "igw-")])
      error_message = "subnet_id must have an Internet Gateway route for temporary EIP setup."

    }

  }
}

resource "aws_security_group" "restore_verification" {
  count       = var.enable_restore_verification ? 1 : 0
  name_prefix = "${local.resource_prefix}-postgres-restore-"
  description = "Isolated PostgreSQL restore verification host"
  vpc_id      = var.vpc_id

  tags = merge(local.common_tags, {
    Component = "database-restore-verification"
  })
}

resource "aws_vpc_security_group_ingress_rule" "restore_verification_ssh" {
  count             = var.enable_restore_verification ? 1 : 0
  security_group_id = aws_security_group.restore_verification[0].id
  cidr_ipv4         = var.operator_ssh_cidr
  ip_protocol       = "tcp"
  from_port         = 22
  to_port           = 22
  description       = "Temporary operator SSH for restore verification"
}

resource "aws_vpc_security_group_egress_rule" "restore_verification_https" {
  count             = var.enable_restore_verification ? 1 : 0
  security_group_id = aws_security_group.restore_verification[0].id
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "tcp"
  from_port         = 443
  to_port           = 443
  description       = "Package and S3 HTTPS access"
}

resource "aws_vpc_security_group_egress_rule" "restore_verification_dns" {
  count             = var.enable_restore_verification ? 1 : 0
  security_group_id = aws_security_group.restore_verification[0].id
  cidr_ipv4         = data.aws_vpc.selected.cidr_block
  ip_protocol       = "udp"
  from_port         = 53
  to_port           = 53
  description       = "VPC DNS resolution"
}

resource "aws_instance" "restore_verification" {
  count                       = var.enable_restore_verification ? 1 : 0
  ami                         = local.database_ami_id
  instance_type               = var.database_instance_type
  key_name                    = var.key_pair_name
  iam_instance_profile        = aws_iam_instance_profile.postgres_restore_verifier.name
  subnet_id                   = var.subnet_id
  vpc_security_group_ids      = [aws_security_group.restore_verification[0].id]
  associate_public_ip_address = false

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

  tags = merge(local.common_tags, {
    Name      = "${local.resource_prefix}-postgres-restore-verification"
    Component = "database-restore-verification"
  })
}

resource "aws_eip" "restore_verification" {
  count  = var.enable_restore_verification ? 1 : 0
  domain = "vpc"

  tags = merge(local.common_tags, {
    Name      = "${local.resource_prefix}-postgres-restore-verification"
    Component = "database-restore-verification"
  })
}

resource "aws_eip_association" "restore_verification" {
  count                = var.enable_restore_verification ? 1 : 0
  network_interface_id = aws_instance.restore_verification[0].primary_network_interface_id
  allocation_id        = aws_eip.restore_verification[0].id
}
resource "aws_eip" "temporary_postgres" {
  count  = var.enable_temporary_public_access ? 1 : 0
  domain = "vpc"
  tags = merge(local.common_tags, {
    Name = "${local.resource_prefix}-postgres-temporary", Component = "database"
  })
}
resource "aws_eip_association" "temporary_postgres" {
  count                = var.enable_temporary_public_access ? 1 : 0
  network_interface_id = aws_instance.postgres.primary_network_interface_id
  allocation_id        = aws_eip.temporary_postgres[0].id
}
