data "aws_ami" "nginx" {
  most_recent      = true
  name_regex       = "^rkcy-nginx-[0-9]{8}-[0-9]{6}$"
  owners           = ["self"]
}

resource "aws_key_pair" "nginx" {
  key_name = "rkcy_${var.cluster}_${var.stack}_nginx"
  public_key = file("${var.ssh_key_path}.pub")
}

data "http" "myip" {
  url = "http://ipv4.icanhazip.com"
}

locals {
  ingress_cidrs = sort(distinct(var.public ? [ var.vpc.cidr_block, "${chomp(data.http.myip.body)}/32"] : [ var.vpc.cidr_block ]))
  ingress_80443_cidrs = sort(distinct(var.public ? [ var.inbound_cidr, "${chomp(data.http.myip.body)}/32"] : [ var.inbound_cidr ]))
}

module "nginx_vm" {
  source = "../vm"

  name = "nginx"
  stack = var.stack
  cluster = {
    name = var.cluster
    aggregate = true
  }
  vpc_id = var.vpc.id
  dns_zone = var.dns_zone

  vms = [for i in range(var.nginx_count) :
    {
      subnet_id = var.subnets[i % length(var.subnets)].id
      ip = cidrhost(var.subnets[i % length(var.subnets)].cidr_block, 80)
    }
  ]

  image_id = data.aws_ami.nginx.id
  instance_type = "m4.large"
  key_name = aws_key_pair.nginx.key_name
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
      name  = "nginx_http"
      cidrs = local.ingress_80443_cidrs
      port  = 80
    },
    {
      name  = "nginx_https"
      cidrs = local.ingress_80443_cidrs
      port  = 443
    },
    {
      name  = "nginx_otelcol"
      cidrs = local.ingress_cidrs
      port  = 4317
    },
    {
      name  = "nginx_jaeger_collector"
      cidrs = local.ingress_cidrs
      port  = 14250
    },
  ]
  out_cidrs = [ "0.0.0.0/0" ]
}

module "nginx_configure" {
  source = "../../../shared/nginx"
  count = var.nginx_count

  hostname = module.nginx_vm.vms[0].hostname
  bastion_ip = var.bastion_ip
  ssh_key_path = var.ssh_key_path
  nginx_ip = module.nginx_vm.vms[count.index].ip
  routes = var.routes
}
