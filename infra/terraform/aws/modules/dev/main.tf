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

variable "stack" {
  type = string
}

variable "vpc" {
  type = any
}

variable "subnet_edge" {
  type = any
}

variable "dns_zone" {
  type = any
}

variable "ssh_key_path" {
  type = string
  default = "~/.ssh/rkcy_id_rsa"
}

data "aws_ami" "dev" {
  most_recent      = true
  name_regex       = "^rkcy-dev-[0-9]{8}-[0-9]{6}$"
  owners           = ["self"]
}

resource "aws_key_pair" "dev" {
  key_name = "rkcy_${var.stack}_dev"
  public_key = file("${var.ssh_key_path}.pub")
}

locals {
  dev_ip = cidrhost(var.subnet_edge[0].cidr_block, 113)
}

resource "aws_security_group" "rkcy_dev" {
  name        = "rkcy_${var.stack}_dev"
  description = "Allow SSH and zookeeper inbound traffic"
  vpc_id      = var.vpc.id

  ingress = [
    {
      cidr_blocks      = [ "0.0.0.0/0", ]
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
      cidr_blocks      = [ "0.0.0.0/0", ]
      description      = ""
      from_port        = 11300
      to_port          = 11399
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
    },
  ]

  egress = [
    {
      cidr_blocks      = [ "0.0.0.0/0", ]
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

resource "aws_network_interface" "dev" {
  subnet_id   = var.subnet_edge[0].id
  private_ips = [local.dev_ip]

  security_groups = [aws_security_group.rkcy_dev.id]
}

resource "aws_instance" "dev" {
  ami = data.aws_ami.dev.id
  instance_type = "m4.large"

  key_name = aws_key_pair.dev.key_name

  network_interface {
    network_interface_id = aws_network_interface.dev.id
    device_index = 0
  }

  credit_specification {
    cpu_credits = "unlimited"
  }

  tags = {
    Name = "rkcy_${var.stack}_inst_dev"
  }
}

resource "aws_eip" "dev" {
  vpc = true

  instance = aws_instance.dev.id
  associate_with_private_ip = local.dev_ip
}

resource "aws_route53_record" "dev_public" {
  zone_id = var.dns_zone.zone_id
  name    = "dev.${var.stack}.${var.dns_zone.name}"
  type    = "A"
  ttl     = "300"
  records = [aws_eip.dev.public_ip]
}

resource "aws_route53_record" "dev_private" {
  zone_id = var.dns_zone.zone_id
  name    = "dev.${var.stack}.local.${var.dns_zone.name}"
  type    = "A"
  ttl     = "300"
  records = [local.dev_ip]
}
