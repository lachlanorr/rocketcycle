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
# Host bastion.rkcy.net
#    User ubuntu
#    IdentityFile ~/.ssh/rkcy_id_rsa
variable "ssh_key_path" {
  type = string
  default = "~/.ssh/rkcy_id_rsa"
}

variable "subnet_count" {
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

resource "aws_security_group" "rkcy_edge" {
  name        = "rkcy_edge"
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

resource "aws_subnet" "rkcy_edge" {
  vpc_id            = aws_vpc.rkcy.id
  cidr_block        = cidrsubnet(aws_vpc.rkcy.cidr_block, 8, 0)
  availability_zone = data.aws_availability_zones.zones.names[0]
  map_public_ip_on_launch = true

  depends_on = [aws_internet_gateway.rkcy]

  tags = {
    Name = "rkcy_edge_sn"
  }
}

resource "aws_subnet" "rkcy_app" {
  count             = var.subnet_count
  vpc_id            = aws_vpc.rkcy.id
  cidr_block        = cidrsubnet(aws_vpc.rkcy.cidr_block, 8, count.index + 1)
  availability_zone = data.aws_availability_zones.zones.names[count.index % length(data.aws_availability_zones.zones.names)]
  map_public_ip_on_launch = true

  depends_on = [aws_internet_gateway.rkcy]

  tags = {
    Name = "rkcy_app${count.index+1}_sn"
  }
}

resource "aws_route_table" "rkcy" {
  vpc_id = aws_vpc.rkcy.id
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.rkcy.id
  }
  tags = {
    Name = "rkcy_rt"
  }
}

resource "aws_route_table_association" "rkcy" {
  subnet_id      = aws_subnet.rkcy_edge.id
  route_table_id = aws_route_table.rkcy.id
}

locals {
  bastion_private_ip = cidrhost(aws_subnet.rkcy_edge.cidr_block, 10)
}

resource "aws_network_interface" "bastion" {
  subnet_id   = aws_subnet.rkcy_edge.id
  private_ips = [local.bastion_private_ip]

  security_groups = [aws_security_group.rkcy_edge.id]
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
  ami = data.aws_ami.bastion.id
  instance_type = "t2.micro"

  key_name = aws_key_pair.bastion.key_name

  network_interface {
    network_interface_id = aws_network_interface.bastion.id
    device_index = 0
  }

  credit_specification {
    cpu_credits = "unlimited"
  }

  tags = {
    Name = "rkcy_inst_bastion"
  }
}

resource "aws_eip" "bastion" {
  vpc = true

  instance = aws_instance.bastion.id
  associate_with_private_ip = local.bastion_private_ip
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
  zone_id = data.aws_route53_zone.rkcy_net.zone_id
  name    = "bastion.${data.aws_route53_zone.rkcy_net.name}"
  type    = "A"
  ttl     = "300"
  records = [aws_eip.bastion.public_ip]
}

resource "aws_route53_record" "bastion_private" {
  zone_id = data.aws_route53_zone.rkcy_net.zone_id
  name    = "bastion.local.${data.aws_route53_zone.rkcy_net.name}"
  type    = "A"
  ttl     = "300"
  records = [local.bastion_private_ip]
}
