locals {
  clusterDot = var.cluster.name != null ? "${var.cluster.name}." : ""
  clusterUnderscore = var.cluster.name != null ? "${var.cluster.name}_" : ""
  fullname = "rkcy_${local.clusterUnderscore}${var.stack}_${var.name}"
  fullnameIdxs = [for i in range(length(var.vms)) : "${local.fullname}_${i}"]
  nameDashes = replace(var.name, "_", "-")
  hostnames = [for i in range(length(var.vms)) : "${local.nameDashes}-${i}.${local.clusterDot}${var.stack}.local.${var.dns_zone.name}"]
  public_hostnames = [for i in range(length(var.vms)) : "${local.nameDashes}-${i}.${local.clusterDot}${var.stack}.${var.dns_zone.name}"]
}

resource "aws_security_group" "vm" {
  name        = local.fullname
  description = local.fullname
  vpc_id      = var.vpc_id

  ingress = [for rule in var.in_rules :
    {
      cidr_blocks      = rule.cidrs
      description      = rule.name
      from_port        = rule.port
      to_port          = rule.port
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
    }
  ]

  egress = [
    {
      cidr_blocks      = var.out_cidrs
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

resource "aws_network_interface" "vm" {
  count = length(var.vms)
  subnet_id   = var.vms[count.index].subnet_id
  private_ips = [var.vms[count.index].ip]

  security_groups = [aws_security_group.vm.id]
}

resource "aws_placement_group" "vm" {
  name     = "${local.fullname}_pg"
  strategy = "spread"
}

resource "aws_instance" "vm" {
  count = length(var.vms)
  ami = var.image_id
  instance_type = var.instance_type
  placement_group = aws_placement_group.vm.name

  key_name = var.key_name

  network_interface {
    network_interface_id = aws_network_interface.vm[count.index].id
    device_index = 0
  }

  credit_specification {
    cpu_credits = "unlimited"
  }

  tags = {
    Name = local.fullnameIdxs[count.index]
  }
}

resource "aws_route53_record" "vm_private" {
  count = length(var.vms)
  zone_id = var.dns_zone.zone_id
  name    = local.hostnames[count.index]
  type    = "A"
  ttl     = "300"
  records = [var.vms[count.index].ip]
}

resource "aws_route53_record" "private_aggregate" {
  count = var.cluster.aggregate ? 1 : 0
  zone_id = var.dns_zone.zone_id
  name    = "${var.cluster.name}.${var.stack}.local.${var.dns_zone.name}"
  type    = "A"
  ttl     = "300"
  records = var.vms[*].ip
}

resource "aws_eip" "vm" {
  count = var.public ? length(var.vms) : 0
  vpc = true

  instance = aws_instance.vm[count.index].id
  associate_with_private_ip = var.vms[count.index].ip

  tags = {
    Name = local.fullnameIdxs[count.index]
  }
}

resource "aws_route53_record" "vm_public" {
  count = var.public ? length(var.vms) : 0
  zone_id = var.dns_zone.zone_id
  name    = local.public_hostnames[count.index]
  type    = "A"
  ttl     = "300"
  records = [aws_eip.vm[count.index].public_ip]
}

resource "aws_route53_record" "public_aggregate" {
  count = var.public && var.cluster.aggregate ? 1 : 0
  zone_id = var.dns_zone.zone_id
  name    = "${var.cluster.name}.${var.stack}.${var.dns_zone.name}"
  type    = "A"
  ttl     = "300"
  records = aws_eip.vm[*].public_ip
}
