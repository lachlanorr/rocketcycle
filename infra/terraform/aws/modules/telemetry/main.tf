data "aws_ami" "telemetry" {
  most_recent      = true
  name_regex       = "^rkcy-telemetry-[0-9]{8}-[0-9]{6}$"
  owners           = ["self"]
}

resource "aws_key_pair" "telemetry" {
  key_name = "rkcy-${var.stack}-telemetry"
  public_key = file("${var.ssh_key_path}.pub")
}

#-------------------------------------------------------------------------------
# jaeger_collector
#-------------------------------------------------------------------------------
module "jaeger_collector_vm" {
  source = "../vm"

  name = "jaeger_collector"
  stack = var.stack
  vpc_id = var.vpc.id
  dns_zone = var.dns_zone

  vms = [for i in range(var.jaeger_collector_count) :
    {
      subnet_id = var.subnets[i % length(var.subnets)].id
      ip = cidrhost(var.subnets[i % length(var.subnets)].cidr_block, 14)
    }
  ]

  image_id = data.aws_ami.telemetry.id
  instance_type = "m4.large"
  key_name = aws_key_pair.telemetry.key_name
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
      name  = "jaeger_collector_grpc"
      cidrs = [ var.vpc.cidr_block ]
      port  = var.jaeger_collector_port
    },
    {
      name  = "jaeger_collector_http"
      cidrs = [ var.vpc.cidr_block ]
      port  = 14268
    },
    {
      name  = "jaeger_collector_admin"
      cidrs = [ var.vpc.cidr_block ]
      port  = 14269
    },
  ]
  out_cidrs = [ var.vpc.cidr_block ]
}

module "jaeger_collector_configure" {
  source = "../../../shared/telemetry/jaeger_collector"
  count = var.jaeger_collector_count

  hostname = module.jaeger_collector_vm.vms[count.index].hostname
  bastion_ip = var.bastion_ip
  ssh_key_path = var.ssh_key_path
  jaeger_collector_ip = module.jaeger_collector_vm.vms[count.index].ip
  elasticsearch_urls = var.elasticsearch_urls
}
#-------------------------------------------------------------------------------
# jaeger_collector (END)
#-------------------------------------------------------------------------------

#-------------------------------------------------------------------------------
# jaeger_query
#-------------------------------------------------------------------------------
module "jaeger_query_vm" {
  source = "../vm"

  name = "jaeger_query"
  stack = var.stack
  vpc_id = var.vpc.id
  dns_zone = var.dns_zone

  vms = [for i in range(var.jaeger_query_count) :
    {
      subnet_id = var.subnets[i % length(var.subnets)].id
      ip = cidrhost(var.subnets[i % length(var.subnets)].cidr_block, 16)
    }
  ]

  image_id = data.aws_ami.telemetry.id
  instance_type = "m4.large"
  key_name = aws_key_pair.telemetry.key_name
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
      name  = "jaeger_query_grpc"
      cidrs = [ var.vpc.cidr_block ]
      port  = 16685
    },
    {
      name  = "jaeger_query_http"
      cidrs = [ var.vpc.cidr_block ]
      port  = var.jaeger_query_port
    },
    {
      name  = "jaeger_query_admin"
      cidrs = [ var.vpc.cidr_block ]
      port  = 16687
    },
  ]
  out_cidrs = [ var.vpc.cidr_block ]
}

module "jaeger_query_configure" {
  source = "../../../shared/telemetry/jaeger_query"
  count = var.jaeger_query_count

  hostname = module.jaeger_query_vm.vms[count.index].hostname
  bastion_ip = var.bastion_ip
  ssh_key_path = var.ssh_key_path
  jaeger_query_ip = module.jaeger_query_vm.vms[count.index].ip
  elasticsearch_urls = var.elasticsearch_urls
}
#-------------------------------------------------------------------------------
# jaeger_query (END)
#-------------------------------------------------------------------------------

#-------------------------------------------------------------------------------
# otelcol
#-------------------------------------------------------------------------------
module "otelcol_vm" {
  source = "../vm"

  name = "otelcol"
  stack = var.stack
  vpc_id = var.vpc.id
  dns_zone = var.dns_zone

  vms = [for i in range(var.otelcol_count) :
   {
      subnet_id = var.subnets[i % length(var.subnets)].id
      ip = cidrhost(var.subnets[i % length(var.subnets)].cidr_block, 43)
    }
  ]

  image_id = data.aws_ami.telemetry.id
  instance_type = "m4.large"
  key_name = aws_key_pair.telemetry.key_name
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
      cidrs = [ var.vpc.cidr_block ]
      name  = "otelcol_grpc"
      port  = var.otelcol_port
    },
    {
      cidrs = [ var.vpc.cidr_block ]
      name  = "otelcol_http"
      port  = 4318
    },
    {
      cidrs = [ var.vpc.cidr_block ]
      name  = "otelcol_prometheus_scrape"
      port  = 8888
    },
    {
      cidrs = [ var.vpc.cidr_block ]
      name  = "otelcol_prometheus_exporter"
      port  = 9999
    },
    {
      cidrs = [ var.vpc.cidr_block ]
      name  = "otelcol_http_legacy"
      port  = 55681
    },
  ]
  out_cidrs = [ var.vpc.cidr_block ]
}

module "otelcol_configure" {
  source = "../../../shared/telemetry/otelcol"
  count = var.otelcol_count

  hostname = module.otelcol_vm.vms[count.index].hostname
  bastion_ip = var.bastion_ip
  ssh_key_path = var.ssh_key_path
  otelcol_ip = module.otelcol_vm.vms[count.index].ip
  elasticsearch_urls = var.elasticsearch_urls
  nginx_telem_host = var.nginx_telem_host
  jaeger_collector_port = var.jaeger_collector_port
}
#-------------------------------------------------------------------------------
# otelcol (END)
#-------------------------------------------------------------------------------
