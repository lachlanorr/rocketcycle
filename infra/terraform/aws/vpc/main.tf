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


resource "aws_vpc" "rkcy" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = "rkcy"
  }
}

resource "aws_security_group" "rkcy_allow_ssh" {
  name        = "rkcy_allow_ssh"
  description = "Allow SSH inbound traffic"
  vpc_id      = aws_vpc.rkcy.id

  ingress = [
    {
#[aws_vpc.rkcy.cidr_block]
      cidr_blocks      = [ "0.0.0.0/0", ]
      description      = ""
      from_port        = 22
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
      to_port          = 22
    }
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

resource "aws_internet_gateway" "rkcy" {
  vpc_id = aws_vpc.rkcy.id

  tags = {
    Name = "rkcy_gw"
  }
}

resource "aws_subnet" "rkcy" {
  vpc_id            = aws_vpc.rkcy.id
  cidr_block        = cidrsubnet(aws_vpc.rkcy.cidr_block, 8, 1)
  availability_zone = "us-east-2a"
  map_public_ip_on_launch = true

  depends_on = [aws_internet_gateway.rkcy]
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
  subnet_id      = aws_subnet.rkcy.id
  route_table_id = aws_route_table.rkcy.id
}

resource "aws_network_interface" "rkcy_bastion" {
  subnet_id   = aws_subnet.rkcy.id
  private_ips = ["10.0.1.100"]

  security_groups = [aws_security_group.rkcy_allow_ssh.id]
}

data "aws_ami" "bastion" {
  most_recent      = true
  name_regex       = "^rkcy-bastion-[0-9]{8}-[0-9]{6}$"
  owners           = ["self"]
}

resource "aws_key_pair" "bastion" {
  key_name = "rkcy-bastion"
  public_key = file("~/.ssh/id_rsa.pub")
}

resource "aws_instance" "bastion" {
  ami = data.aws_ami.bastion.id
  instance_type = "t2.micro"

  key_name = aws_key_pair.bastion.key_name

  network_interface {
    network_interface_id = aws_network_interface.rkcy_bastion.id
    device_index = 0
  }

  credit_specification {
    cpu_credits = "unlimited"
  }

  tags = {
    Name = "rkcy_bastion"
  }
}

resource "aws_eip" "bastion" {
  vpc = true

  instance = aws_instance.bastion.id
  associate_with_private_ip = tolist(aws_network_interface.rkcy_bastion.private_ips)[0]
  depends_on = [aws_internet_gateway.rkcy]
}

data "aws_route53_zone" "rkcy_net" {
  name         = "rkcy.net"
}

resource "aws_route53_record" "bastion" {
  zone_id = data.aws_route53_zone.rkcy_net.zone_id
  name    = "bastion.${data.aws_route53_zone.rkcy_net.name}"
  type    = "A"
  ttl     = "300"
  records = [aws_eip.bastion.public_ip]
}
