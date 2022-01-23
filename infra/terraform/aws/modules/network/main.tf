resource "aws_vpc" "rkcy" {
  cidr_block = var.cidr_block

  tags = {
    Name = "rkcy_${var.stack}_vpc"
  }
}

data "http" "myip" {
  url = "http://ipv4.icanhazip.com"
}

resource "aws_security_group" "rkcy_bastion" {
  name        = "rkcy_${var.stack}_bastion"
  description = "Allow SSH inbound traffic"
  vpc_id      = aws_vpc.rkcy.id

  ingress = [
    {
      cidr_blocks      = [ "${chomp(data.http.myip.body)}/32" ]
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
      cidr_blocks      = [ aws_vpc.rkcy.cidr_block ]
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
    Name = "rkcy_${var.stack}_gw"
  }
}

data "aws_availability_zones" "azs" {
  state = "available"
  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

locals {
  azs = sort(data.aws_availability_zones.azs.names)
}

resource "aws_subnet" "rkcy_edge" {
  count             = var.edge_subnet_count
  vpc_id            = aws_vpc.rkcy.id
  cidr_block        = cidrsubnet(aws_vpc.rkcy.cidr_block, 8, 0 + count.index)
  availability_zone = local.azs[count.index % length(local.azs)]
  map_public_ip_on_launch = false

  depends_on = [aws_internet_gateway.rkcy]

  tags = {
    Name = "rkcy_${var.stack}_edge_${count.index}_sn"
  }
}

resource "aws_subnet" "rkcy_app" {
  count             = var.app_subnet_count
  vpc_id            = aws_vpc.rkcy.id
  cidr_block        = cidrsubnet(aws_vpc.rkcy.cidr_block, 8, 100 + count.index)
  availability_zone = local.azs[count.index % length(local.azs)]
  map_public_ip_on_launch = false

  depends_on = [aws_internet_gateway.rkcy]

  tags = {
    Name = "rkcy_${var.stack}_app_${count.index}_sn"
  }
}

resource "aws_subnet" "rkcy_storage" {
  count             = var.storage_subnet_count
  vpc_id            = aws_vpc.rkcy.id
  cidr_block        = cidrsubnet(aws_vpc.rkcy.cidr_block, 8, 200 + count.index)
  availability_zone = local.azs[count.index % length(local.azs)]
  map_public_ip_on_launch = false

  depends_on = [aws_internet_gateway.rkcy]

  tags = {
    Name = "rkcy_${var.stack}_storage_${count.index}_sn"
  }
}

resource "aws_route_table" "rkcy" {
  vpc_id = aws_vpc.rkcy.id
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.rkcy.id
  }
  tags = {
    Name = "rkcy_${var.stack}_rt"
  }
}

resource "aws_route_table_association" "rkcy_edge" {
  count          = var.edge_subnet_count
  subnet_id      = aws_subnet.rkcy_edge[count.index].id
  route_table_id = aws_route_table.rkcy.id
}

resource "aws_route_table_association" "rkcy_app" {
  count          = var.app_subnet_count
  subnet_id      = aws_subnet.rkcy_app[count.index].id
  route_table_id = aws_route_table.rkcy.id
}

resource "aws_route_table_association" "rkcy_storage" {
  count          = var.storage_subnet_count
  subnet_id      = aws_subnet.rkcy_storage[count.index].id
  route_table_id = aws_route_table.rkcy.id
}

locals {
  bastion_private_ips = [for i in range(var.bastion_count) : "${cidrhost(aws_subnet.rkcy_edge[i].cidr_block, 10)}" ]
}

resource "aws_network_interface" "bastion" {
  count       = var.bastion_count
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
  key_name = "rkcy_${var.stack}_bastion"
  public_key = file("${var.ssh_key_path}.pub")
}

resource "aws_instance" "bastion" {
  count = var.bastion_count
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
    Name = "rkcy_${var.stack}_bastion_${count.index}"
  }
}

resource "aws_eip" "bastion" {
  count = var.bastion_count
  vpc = true

  instance = aws_instance.bastion[count.index].id
  associate_with_private_ip = local.bastion_private_ips[count.index]
  depends_on = [aws_internet_gateway.rkcy]

  tags = {
    Name = "rkcy_${var.stack}_eip_bastion_${count.index}"
  }
}

data "aws_route53_zone" "zone" {
  name = var.dns_zone
}

resource "aws_route53_record" "bastion_public" {
  count   = var.bastion_count
  zone_id = data.aws_route53_zone.zone.zone_id
  name    = "bastion-${count.index}.${var.stack}.${data.aws_route53_zone.zone.name}"
  type    = "A"
  ttl     = "300"
  records = [aws_eip.bastion[count.index].public_ip]
}

resource "aws_route53_record" "bastion_private" {
  count   = var.bastion_count
  zone_id = data.aws_route53_zone.zone.zone_id
  name    = "bastion-${count.index}.${var.stack}.local.${data.aws_route53_zone.zone.name}"
  type    = "A"
  ttl     = "300"
  records = [local.bastion_private_ips[count.index]]
}

module "bastion_configure" {
  source = "../../../shared/network"
  count = var.bastion_count
  depends_on = [aws_instance.bastion]

  hostname = aws_route53_record.bastion_private[count.index].name
  bastion_ip = aws_eip.bastion[count.index].public_ip
  ssh_key_path = var.ssh_key_path
}
