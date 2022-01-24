locals {
  sn_ids   = var.subnet_storage.*.id
  sn_cidrs = var.subnet_storage.*.cidr_block
}

data "aws_ami" "elasticsearch" {
  most_recent      = true
  name_regex       = "^rkcy-elasticsearch-[0-9]{8}-[0-9]{6}$"
  owners           = ["self"]
}

resource "aws_key_pair" "elasticsearch" {
  key_name = "rkcy-${var.stack}-elasticsearch"
  public_key = file("${var.ssh_key_path}.pub")
}

locals {
  elasticsearch_racks = [for i in range(var.elasticsearch_count) : "${var.azs[i % var.elasticsearch_count]}"]
  elasticsearch_ips = [for i in range(var.elasticsearch_count) : "${cidrhost(local.sn_cidrs[i], 92)}"]
  elasticsearch_nodes = [for i in range(var.elasticsearch_count) : "master-${i}"]
}

resource "aws_security_group" "rkcy_elasticsearch" {
  name        = "rkcy_${var.stack}_elasticsearch"
  description = "Allow SSH and elasticsearch inbound traffic"
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
      from_port        = var.elasticsearch_port
      to_port          = var.elasticsearch_port
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
    },
    {
      cidr_blocks      = [ var.vpc.cidr_block ]
      description      = ""
      from_port        = 9300
      to_port          = 9300
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

resource "aws_network_interface" "elasticsearch" {
  count = var.elasticsearch_count
  subnet_id   = local.sn_ids[count.index]
  private_ips = [local.elasticsearch_ips[count.index]]

  security_groups = [aws_security_group.rkcy_elasticsearch.id]
}

resource "aws_placement_group" "elasticsearch" {
  name     = "rkcy_${var.stack}_elasticsearch_pc"
  strategy = "spread"
}

resource "aws_instance" "elasticsearch" {
  count = var.elasticsearch_count
  ami = data.aws_ami.elasticsearch.id
  instance_type = "m4.large"
  placement_group = aws_placement_group.elasticsearch.name

  key_name = aws_key_pair.elasticsearch.key_name

  network_interface {
    network_interface_id = aws_network_interface.elasticsearch[count.index].id
    device_index = 0
  }

  credit_specification {
    cpu_credits = "unlimited"
  }
  tags = {
    Name = "rkcy_${var.stack}_elasticsearch_${count.index}"
  }
}

resource "aws_route53_record" "elasticsearch_private" {
  count = var.elasticsearch_count
  zone_id = var.dns_zone.zone_id
  name    = "elasticsearch-${count.index}.${var.stack}.local.${var.dns_zone.name}"
  type    = "A"
  ttl     = "300"
  records = [local.elasticsearch_ips[count.index]]
}

module "elasticsearch_configure" {
  source = "../../../shared/elasticsearch"
  count = var.elasticsearch_count
  depends_on = [aws_instance.elasticsearch]

  hostname = aws_route53_record.elasticsearch_private[count.index].name
  bastion_ip = var.bastion_ips[0]
  ssh_key_path = var.ssh_key_path
  stack = var.stack
  elasticsearch_index = count.index
  elasticsearch_ips = local.elasticsearch_ips
  elasticsearch_nodes = local.elasticsearch_nodes
  elasticsearch_rack = local.elasticsearch_racks[count.index]
}
