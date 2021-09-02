terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
      version = "~> 3.27"
    }
  }

  required_version = ">= 0.14.9"
}

provider "aws" {
  profile = "default"
  region = "us-east-2"
}

variable "vpc_cidr_block" {
  type = string
  default = "10.0.0.0/16"
}

# Put something like this in ~/.ssh/config:
#
# Host bastion0.rkcy.net
#    User ubuntu
#    IdentityFile ~/.ssh/rkcy_id_rsa
variable "ssh_key_path" {
  type = string
  default = "~/.ssh/rkcy_id_rsa"
}

variable "edge_subnet_count" {
  type = number
  default = 3
}

variable "app_subnet_count" {
  type = number
  default = 5
}

variable "storage_subnet_count" {
  type = number
  default = 5
}

resource "aws_vpc" "rkcy" {
  cidr_block = var.vpc_cidr_block

  tags = {
    Name = "rkcy_vpc"
  }
}

data "http" "myip" {
  url = "http://ipv4.icanhazip.com"
}

locals {
  allowed_inbound_cidr = "${chomp(data.http.myip.body)}/32"
}

resource "aws_security_group" "rkcy_bastion" {
  name        = "rkcy_bastion"
  description = "Allow SSH inbound traffic"
  vpc_id      = aws_vpc.rkcy.id

  ingress = [
    {
      cidr_blocks      = [ local.allowed_inbound_cidr ]
      description      = ""
      from_port        = 22
      to_port          = 22
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
    }
  ]

  egress = [
    {
      cidr_blocks      = [ "0.0.0.0/0", ]
      description      = ""
      from_port        = 0
      to_port          = 0
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "-1"
      security_groups  = []
      self             = false
    }
  ]
}

resource "aws_internet_gateway" "rkcy" {
  vpc_id = aws_vpc.rkcy.id

  tags = {
    Name = "rkcy_gw"
  }
}

data "aws_availability_zones" "zones" {
  state = "available"
  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

locals {
  zones = sort(data.aws_availability_zones.zones.names)
}

resource "aws_subnet" "rkcy_edge" {
  count             = var.edge_subnet_count
  vpc_id            = aws_vpc.rkcy.id
  cidr_block        = cidrsubnet(aws_vpc.rkcy.cidr_block, 8, 0 + count.index)
  availability_zone = local.zones[count.index % length(local.zones)]
  map_public_ip_on_launch = false

  depends_on = [aws_internet_gateway.rkcy]

  tags = {
    Name = "rkcy_edge${count.index}_sn"
  }
}

resource "aws_subnet" "rkcy_app" {
  count             = var.app_subnet_count
  vpc_id            = aws_vpc.rkcy.id
  cidr_block        = cidrsubnet(aws_vpc.rkcy.cidr_block, 8, 100 + count.index)
  availability_zone = local.zones[count.index % length(local.zones)]
  map_public_ip_on_launch = false

  depends_on = [aws_internet_gateway.rkcy]

  tags = {
    Name = "rkcy_app${count.index}_sn"
  }
}

resource "aws_subnet" "rkcy_storage" {
  count             = var.storage_subnet_count
  vpc_id            = aws_vpc.rkcy.id
  cidr_block        = cidrsubnet(aws_vpc.rkcy.cidr_block, 8, 200 + count.index)
  availability_zone = local.zones[count.index % length(local.zones)]
  map_public_ip_on_launch = false

  depends_on = [aws_internet_gateway.rkcy]

  tags = {
    Name = "rkcy_storage${count.index}_sn"
  }
}

resource "aws_route_table" "rkcy_edge" {
  vpc_id = aws_vpc.rkcy.id
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.rkcy.id
  }
  tags = {
    Name = "rkcy_rt"
  }
}

resource "aws_route_table_association" "rkcy_edge" {
  count          = var.edge_subnet_count
  subnet_id      = aws_subnet.rkcy_edge[count.index].id
  route_table_id = aws_route_table.rkcy_edge.id
}

locals {
  bastion_private_ips = [for i in range(var.edge_subnet_count) : "${cidrhost(aws_subnet.rkcy_edge[i].cidr_block, 10)}" ]
}

resource "aws_network_interface" "bastion" {
  count       = var.edge_subnet_count
  subnet_id   = aws_subnet.rkcy_edge[count.index].id
  private_ips = [local.bastion_private_ips[count.index]]

  security_groups = [aws_security_group.rkcy_bastion.id]
}

data "aws_ami" "bastion" {
  most_recent      = true
  name_regex       = "^rkcy-bastion-[0-9]{8}-[0-9]{6}$"
  owners           = ["self"]
}

resource "aws_key_pair" "bastion" {
  key_name = "rkcy-bastion"
  public_key = file("${var.ssh_key_path}.pub")
}

resource "aws_instance" "bastion" {
  count = var.edge_subnet_count
  ami = data.aws_ami.bastion.id
  instance_type = "t2.micro"

  key_name = aws_key_pair.bastion.key_name

  network_interface {
    network_interface_id = aws_network_interface.bastion[count.index].id
    device_index = 0
  }

  credit_specification {
    cpu_credits = "unlimited"
  }

  tags = {
    Name = "rkcy_inst_bastion${count.index}"
  }
}

resource "aws_eip" "bastion" {
  count = var.edge_subnet_count
  vpc = true

  instance = aws_instance.bastion[count.index].id
  associate_with_private_ip = local.bastion_private_ips[count.index]
  depends_on = [aws_internet_gateway.rkcy]

  provisioner "file" {
    source = var.ssh_key_path
    destination = "~/.ssh/id_rsa"
  }

  provisioner "remote-exec" {
    inline = [
      "chmod 600 ~/.ssh/id_rsa"
    ]
  }

  connection {
    type     = "ssh"
    user     = "ubuntu"
    host     = self.public_ip
    private_key = file(var.ssh_key_path)
  }
}

data "aws_route53_zone" "rkcy_net" {
  name = "rkcy.net"
}

resource "aws_route53_record" "bastion_public" {
  count   = var.edge_subnet_count
  zone_id = data.aws_route53_zone.rkcy_net.zone_id
  name    = "bastion${count.index}.${data.aws_route53_zone.rkcy_net.name}"
  type    = "A"
  ttl     = "300"
  records = [aws_eip.bastion[count.index].public_ip]
}

resource "aws_route53_record" "bastion_private" {
  count   = var.edge_subnet_count
  zone_id = data.aws_route53_zone.rkcy_net.zone_id
  name    = "bastion${count.index}.local.${data.aws_route53_zone.rkcy_net.name}"
  type    = "A"
  ttl     = "300"
  records = [local.bastion_private_ips[count.index]]
}
