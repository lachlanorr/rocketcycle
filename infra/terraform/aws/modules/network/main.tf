resource "aws_vpc" "rkcy" {
  cidr_block = var.cidr_block

  tags = {
    Name = "rkcy_${var.stack}_vpc"
  }
}

data "http" "myip" {
  url = "http://ipv4.icanhazip.com"
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

data "aws_route53_zone" "zone" {
  name = var.dns_zone
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

module "bastion_vm" {
  source = "../vm"

  name = "bastion"
  stack = var.stack
  vpc_id = aws_vpc.rkcy.id
  dns_zone = data.aws_route53_zone.zone

  vms = [for i in range(var.bastion_count) :
    {
      subnet_id = aws_subnet.rkcy_edge[i % length(aws_subnet.rkcy_edge)].id
      ip = cidrhost(aws_subnet.rkcy_edge[i].cidr_block, 10)
    }
  ]

  image_id = data.aws_ami.bastion.id
  instance_type = "t2.micro"
  key_name = aws_key_pair.bastion.key_name
  public = true

  in_rules = [
    {
      name  = "ssh"
      cidrs = [ "${chomp(data.http.myip.body)}/32", aws_vpc.rkcy.cidr_block ]
      port  = 22
    },
    {
      name  = "node_exporter"
      cidrs = [ aws_vpc.rkcy.cidr_block ]
      port  = 9100
    },
  ]
  out_cidrs = [ "0.0.0.0/0" ]
}

module "bastion_configure" {
  source = "../../../shared/network"
  count = var.bastion_count

  hostname = module.bastion_vm.vms[count.index].hostname
  bastion_ip = module.bastion_vm.vms[count.index].public_ip
  ssh_key_path = var.ssh_key_path
}
