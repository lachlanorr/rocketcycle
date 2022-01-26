data "aws_ami" "metrics" {
  most_recent      = true
  name_regex       = "^rkcy-metrics-[0-9]{8}-[0-9]{6}$"
  owners           = ["self"]
}

resource "aws_key_pair" "metrics" {
  key_name = "rkcy-${var.stack}-metrics"
  public_key = file("${var.ssh_key_path}.pub")
}

#-------------------------------------------------------------------------------
# prometheus
#-------------------------------------------------------------------------------
module "prometheus_vm" {
  source = "../vm"

  name = "prometheus"
  stack = var.stack
  vpc_id = var.vpc.id
  dns_zone = var.dns_zone

  vms = [for i in range(var.prometheus_count) :
    {
      subnet_id = var.subnets[i % length(var.subnets)].id
      ip = cidrhost(var.subnets[i % length(var.subnets)].cidr_block, 90)
    }
  ]

  image_id = data.aws_ami.metrics.id
  instance_type = "m4.large"
  key_name = aws_key_pair.metrics.key_name
  public = false

  in_rules = [
    {
      name  = "ssh"
      cidrs = [ var.vpc.cidr_block ]
      port  = 22
    },
    {
      name  = "node_exporter"
      cidrs = [ var.vpc.cidr_block ]
      port  = 9100
    },
    {
      name  = "prometheus"
      cidrs = [ var.vpc.cidr_block ]
      port  = var.prometheus_port
    },
  ]
  out_cidrs = [ var.vpc.cidr_block ]
}

module "prometheus_configure" {
  source = "../../../shared/metrics/prometheus"
  count = var.prometheus_count

  hostname = module.prometheus_vm.vms[count.index].hostname
  bastion_ip = var.bastion_ip
  ssh_key_path = var.ssh_key_path

  jobs = var.jobs
  balancer_url = var.balancer_external_urls.edge
  prometheus_port = var.prometheus_port
  prometheus_ip = module.prometheus_vm.vms[count.index].ip
  prometheus_hosts = sort(module.prometheus_vm.vms[*].hostname)
  grafana_hosts = sort(module.grafana_vm.vms[*].hostname)
}
#-------------------------------------------------------------------------------
# prometheus (END)
#-------------------------------------------------------------------------------

#-------------------------------------------------------------------------------
# grafana
#-------------------------------------------------------------------------------
module "grafana_vm" {
  source = "../vm"

  name = "grafana"
  stack = var.stack
  vpc_id = var.vpc.id
  dns_zone = var.dns_zone

  vms = [for i in range(var.grafana_count) :
    {
      subnet_id = var.subnets[i % length(var.subnets)].id
      ip = cidrhost(var.subnets[i % length(var.subnets)].cidr_block, 30)
    }
  ]

  image_id = data.aws_ami.metrics.id
  instance_type = "m4.large"
  key_name = aws_key_pair.metrics.key_name
  public = false

  in_rules = [
    {
      name  = "ssh"
      cidrs = [ var.vpc.cidr_block ]
      port  = 22
    },
    {
      name  = "node_exporter"
      cidrs = [ var.vpc.cidr_block ]
      port  = 9100
    },
    {
      name  = "grafana"
      cidrs = [ var.vpc.cidr_block ]
      port  = var.grafana_port
    },
  ]
  out_cidrs = [ var.vpc.cidr_block ]
}

module "grafana_configure" {
  source = "../../../shared/metrics/grafana"
  count = var.grafana_count

  hostname = module.grafana_vm.vms[count.index].hostname
  bastion_ip = var.bastion_ip
  ssh_key_path = var.ssh_key_path

  balancer_external_url = var.balancer_external_urls.edge
  balancer_internal_url = var.balancer_internal_urls.app
  grafana_port = var.grafana_port
  grafana_ip = module.grafana_vm.vms[count.index].ip
}
#-------------------------------------------------------------------------------
# grafana (END)
#-------------------------------------------------------------------------------
