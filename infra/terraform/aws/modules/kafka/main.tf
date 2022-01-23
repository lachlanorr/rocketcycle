locals {
  sn_ids   = var.subnet_app.*.id
  sn_cidrs = var.subnet_app.*.cidr_block
}

data "aws_ami" "kafka" {
  most_recent      = true
  name_regex       = "^rkcy-kafka-[0-9]{8}-[0-9]{6}$"
  owners           = ["self"]
}

resource "aws_key_pair" "kafka" {
  key_name = "rkcy-${var.cluster}-${var.stack}-kafka"
  public_key = file("${var.ssh_key_path}.pub")
}

data "http" "myip" {
  url = "http://ipv4.icanhazip.com"
}

locals {
  ingress_cidrs = var.public ? [ var.vpc.cidr_block, "${chomp(data.http.myip.body)}/32"] : [ var.vpc.cidr_block ]
  egress_cidrs = var.public ? [ "0.0.0.0/0" ] : [ var.vpc.cidr_block ]
}

#-------------------------------------------------------------------------------
# Zookeepers
#-------------------------------------------------------------------------------
locals {
  zookeeper_ips = [for i in range(var.zookeeper_count) : "${cidrhost(local.sn_cidrs[i], 100)}"]
}

resource "aws_security_group" "rkcy_zookeeper" {
  name        = "rkcy_${var.cluster}_${var.stack}_zookeeper"
  description = "Allow SSH and zookeeper inbound traffic"
  vpc_id      = var.vpc.id

  ingress = [
    {
      cidr_blocks      = [ var.vpc.cidr_block ]
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
      cidr_blocks      = [ var.vpc.cidr_block ]
      description      = ""
      from_port        = 2181
      to_port          = 2181
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
    },
    {
      cidr_blocks      = [ var.vpc.cidr_block ]
      description      = ""
      from_port        = 2888
      to_port          = 2888
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
    },
    {
      cidr_blocks      = [ var.vpc.cidr_block ]
      description      = ""
      from_port        = 3888
      to_port          = 3888
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
    },
    {
      cidr_blocks      = [ var.vpc.cidr_block ]
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
      cidr_blocks      = [ var.vpc.cidr_block ]
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

resource "aws_network_interface" "zookeeper" {
  count = var.zookeeper_count
  subnet_id   = local.sn_ids[count.index]
  private_ips = [local.zookeeper_ips[count.index]]

  security_groups = [aws_security_group.rkcy_zookeeper.id]
}

resource "aws_placement_group" "zookeeper" {
  name     = "rkcy_${var.cluster}_${var.stack}_zookeeper_pc"
  strategy = "spread"
}

resource "aws_instance" "zookeeper" {
  count = var.zookeeper_count
  ami = data.aws_ami.kafka.id
  instance_type = "m4.large"
  placement_group = aws_placement_group.zookeeper.name

  key_name = aws_key_pair.kafka.key_name

  network_interface {
    network_interface_id = aws_network_interface.zookeeper[count.index].id
    device_index = 0
  }

  credit_specification {
    cpu_credits = "unlimited"
  }

  tags = {
    Name = "rkcy_${var.cluster}_${var.stack}_zookeeper_${count.index}"
  }
}

resource "aws_route53_record" "zookeeper_private" {
  count = var.zookeeper_count
  zone_id = var.dns_zone.zone_id
  name    = "zookeeper-${count.index}.${var.cluster}.${var.stack}.local.${var.dns_zone.name}"
  type    = "A"
  ttl     = "300"
  records = [local.zookeeper_ips[count.index]]
}

module "zookeeper_configure" {
  source = "../../../shared/kafka/zookeeper"
  count = var.zookeeper_count
  depends_on = [aws_instance.zookeeper]

  hostname = aws_route53_record.zookeeper_private[count.index].name
  bastion_ip = var.bastion_ips[0]
  ssh_key_path = var.ssh_key_path
  zookeeper_index = count.index
  zookeeper_ips = local.zookeeper_ips
}
#-------------------------------------------------------------------------------
# Zookeepers (END)
#-------------------------------------------------------------------------------


#-------------------------------------------------------------------------------
# Brokers
#-------------------------------------------------------------------------------
locals {
  kafka_racks = [for i in range(var.kafka_count) : "${var.azs[i % length(var.azs)]}"]
  kafka_internal_ips = [for i in range(var.kafka_count) : "${cidrhost(local.sn_cidrs[i], 101)}"]
  kafka_internal_hosts = [for i in range(var.kafka_count) : "kafka-${i}.${var.cluster}.${var.stack}.local.${var.dns_zone.name}"]
  kafka_external_ips = aws_eip.kafka.*.public_ip
  kafka_external_hosts = [for i in range(var.kafka_count) : "kafka-${i}.${var.cluster}.${var.stack}.${var.dns_zone.name}"]
}

resource "aws_security_group" "rkcy_kafka" {
  name        = "rkcy_${var.cluster}_${var.stack}_kafka"
  description = "Allow SSH and kafka inbound traffic"
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
      from_port        = 9092
      to_port          = 9093
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

resource "aws_network_interface" "kafka" {
  count = var.kafka_count
  subnet_id   = local.sn_ids[count.index]
  private_ips = [local.kafka_internal_ips[count.index]]

  security_groups = [aws_security_group.rkcy_kafka.id]
}

resource "aws_placement_group" "kafka" {
  name     = "rkcy_${var.cluster}_${var.stack}_kafka_pc"
  strategy = "spread"
}

resource "aws_instance" "kafka" {
  count = var.kafka_count
  ami = data.aws_ami.kafka.id
  instance_type = "m4.large"
  placement_group = aws_placement_group.kafka.name

  key_name = aws_key_pair.kafka.key_name

  network_interface {
    network_interface_id = aws_network_interface.kafka[count.index].id
    device_index = 0
  }

  credit_specification {
    cpu_credits = "unlimited"
  }

  tags = {
    Name = "rkcy_${var.cluster}_${var.stack}_kafka_${count.index}"
  }
}

resource "aws_eip" "kafka" {
  count = var.public ? var.kafka_count : 0
  vpc = true

  instance = aws_instance.kafka[count.index].id
  associate_with_private_ip = local.kafka_internal_ips[count.index]

  tags = {
    Name = "rkcy_${var.cluster}_${var.stack}_eip_kafka_${count.index}"
  }
}

resource "aws_route53_record" "kafka_public" {
  count   = var.public ? var.kafka_count : 0
  zone_id = var.dns_zone.zone_id
  name    = local.kafka_external_hosts[count.index]
  type    = "A"
  ttl     = "300"
  records = [aws_eip.kafka[count.index].public_ip]
}

resource "aws_route53_record" "kafka_private" {
  count = var.kafka_count
  zone_id = var.dns_zone.zone_id
  name    = local.kafka_internal_hosts[count.index]
  type    = "A"
  ttl     = "300"
  records = [local.kafka_internal_ips[count.index]]
}

module "kafka_configure" {
  source = "../../../shared/kafka"
  count = var.kafka_count
  depends_on = [
    aws_instance.kafka,
    module.zookeeper_configure,
  ]

  hostname = aws_route53_record.kafka_private[count.index].name
  bastion_ip = var.bastion_ips[0]
  ssh_key_path = var.ssh_key_path
  public = var.public

  kafka_index = count.index
  kafka_rack = local.kafka_racks[count.index]
  kafka_internal_ips = local.kafka_internal_ips
  kafka_internal_host = local.kafka_internal_hosts[count.index]
  kafka_external_host = local.kafka_external_hosts[count.index]

  zookeeper_ips = local.zookeeper_ips
}
#-------------------------------------------------------------------------------
# Brokers (END)
#-------------------------------------------------------------------------------
