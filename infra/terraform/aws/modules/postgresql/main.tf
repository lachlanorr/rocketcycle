locals {
  sn_ids   = "${values(zipmap(var.subnet_storage.*.cidr_block, var.subnet_storage.*.id))}"
  sn_cidrs = "${values(zipmap(var.subnet_storage.*.cidr_block, var.subnet_storage.*.cidr_block))}"
}

data "aws_ami" "postgresql" {
  most_recent      = true
  name_regex       = "^rkcy-postgresql-[0-9]{8}-[0-9]{6}$"
  owners           = ["self"]
}

resource "aws_key_pair" "postgresql" {
  key_name = "rkcy-${var.stack}-postgresql"
  public_key = file("${var.ssh_key_path}.pub")
}

data "http" "myip" {
  url = "http://ipv4.icanhazip.com"
}

locals {
  ingress_cidrs = var.public ? [ var.vpc.cidr_block, "${chomp(data.http.myip.body)}/32"] : [ var.vpc.cidr_block ]
  egress_cidrs = var.public ? [ "0.0.0.0/0" ] : [ var.vpc.cidr_block ]
}

locals {
  postgresql_ips = [for i in range(var.postgresql_count) : "${cidrhost(local.sn_cidrs[i], 100)}"]
}

resource "aws_security_group" "rkcy_postgresql" {
  name        = "rkcy_${var.stack}_postgresql"
  description = "Allow SSH and postgresql inbound traffic"
  vpc_id      = var.vpc.id

  ingress = [
    {
      cidr_blocks      = local.ingress_cidrs
      description      = ""
      from_port        = 22
      to_port          = 22
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
    },
    {
      cidr_blocks      = local.ingress_cidrs
      description      = ""
      from_port        = 5432
      to_port          = 5432
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
    },
    {
      cidr_blocks      = local.ingress_cidrs
      description      = "node_exporter"
      from_port        = 9100
      to_port          = 9100
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
    },
  ]

  egress = [
    {
      cidr_blocks      = local.egress_cidrs
      description      = ""
      from_port        = 0
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "-1"
      security_groups  = []
      self             = false
      to_port          = 0
    }
  ]
}

resource "aws_network_interface" "postgresql" {
  count = var.postgresql_count
  subnet_id   = local.sn_ids[count.index]
  private_ips = [local.postgresql_ips[count.index]]

  security_groups = [aws_security_group.rkcy_postgresql.id]
}

resource "aws_placement_group" "postgresql" {
  name     = "rkcy_${var.stack}_postgresql_pc"
  strategy = "spread"
}

resource "aws_instance" "postgresql" {
  count = var.postgresql_count
  ami = data.aws_ami.postgresql.id
  instance_type = "m4.large"
  placement_group = aws_placement_group.postgresql.name

  key_name = aws_key_pair.postgresql.key_name

  network_interface {
    network_interface_id = aws_network_interface.postgresql[count.index].id
    device_index = 0
  }

  credit_specification {
    cpu_credits = "unlimited"
  }

  tags = {
    Name = "rkcy_${var.stack}_postgresql_${count.index}"
  }
}

resource "aws_eip" "postgresql" {
  count = var.public ? var.postgresql_count : 0
  vpc = true

  instance = aws_instance.postgresql[count.index].id
  associate_with_private_ip = local.postgresql_ips[count.index]

  tags = {
    Name = "rkcy_${var.stack}_eip_postgresql_${count.index}"
  }
}

resource "aws_route53_record" "postgresql_public" {
  count = var.public ? var.postgresql_count : 0
  zone_id = var.dns_zone.zone_id
  name    = "postgresql-${count.index}.${var.stack}.${var.dns_zone.name}"
  type    = "A"
  ttl     = "300"
  records = [aws_eip.postgresql[count.index].public_ip]
}

resource "aws_route53_record" "postgresql_private" {
  count = var.postgresql_count
  zone_id = var.dns_zone.zone_id
  name    = "postgresql-${count.index}.${var.stack}.local.${var.dns_zone.name}"
  type    = "A"
  ttl     = "300"
  records = [local.postgresql_ips[count.index]]
}

module "postgresql_configure" {
  source = "../../../shared/postgresql"
  count = var.postgresql_count
  depends_on = [aws_instance.postgresql]

  hostname = aws_route53_record.postgresql_private[count.index].name
  bastion_ip = var.bastion_ips[0]
  ssh_key_path = var.ssh_key_path
  postgresql_ip = local.postgresql_ips[count.index]
}
