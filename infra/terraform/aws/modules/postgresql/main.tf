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

module "postgresql_vm" {
  source = "../vm"

  name = "postgresql"
  stack = var.stack
  vpc_id = var.vpc.id
  dns_zone = var.dns_zone

  vms = [for i in range(var.postgresql_count) :
    {
      subnet_id = var.subnets[i % length(var.subnets)].id
      ip = cidrhost(var.subnets[i % length(var.subnets)].cidr_block, 54)
    }
  ]

  image_id = data.aws_ami.postgresql.id
  instance_type = "m4.large"
  key_name = aws_key_pair.postgresql.key_name
  public = var.public

  in_rules = [
    {
      name  = "ssh"
      cidrs = local.ingress_cidrs
      port  = 22
    },
    {
      name  = "node_exporter"
      cidrs = local.ingress_cidrs
      port  = 9100
    },
    {
      name  = "postgresql"
      cidrs = local.ingress_cidrs
      port  = 5432
    },
  ]
  out_cidrs = local.egress_cidrs
}

module "postgresql_configure" {
  source = "../../../shared/postgresql"
  count = var.postgresql_count

  hostname = module.postgresql_vm.vms[count.index].hostname
  bastion_ip = var.bastion_ip
  ssh_key_path = var.ssh_key_path
  postgresql_ip = module.postgresql_vm.vms[count.index].ip
}
