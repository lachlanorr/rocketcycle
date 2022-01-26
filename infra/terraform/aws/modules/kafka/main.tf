data "aws_ami" "kafka" {
  most_recent      = true
  name_regex       = "^rkcy-kafka-[0-9]{8}-[0-9]{6}$"
  owners           = ["self"]
}

resource "aws_key_pair" "kafka" {
  key_name = "rkcy-${var.cluster}-${var.stack}-kafka"
  public_key = file("${var.ssh_key_path}.pub")
}

data "http" "myip" {
  url = "http://ipv4.icanhazip.com"
}

#-------------------------------------------------------------------------------
# Zookeepers
#-------------------------------------------------------------------------------
module "zookeeper_vm" {
  source = "../vm"

  name = "zookeeper"
  stack = var.stack
  cluster = {
    name = var.cluster
    aggregate = false
  }
  vpc_id = var.vpc.id
  dns_zone = var.dns_zone

  vms = [for i in range(var.zookeeper_count) :
    {
      subnet_id = var.subnets[i % length(var.subnets)].id
      ip = cidrhost(var.subnets[i % length(var.subnets)].cidr_block, 100)
    }
  ]

  image_id = data.aws_ami.kafka.id
  instance_type = "m4.large"
  key_name = aws_key_pair.kafka.key_name
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
      name  = "zookeeper_client"
      cidrs = [ var.vpc.cidr_block ]
      port  = 2181
    },
    {
      name  = "zookeeper_peer"
      cidrs = [ var.vpc.cidr_block ]
      port  = 2888
    },
    {
      name  = "zookeeper_leader"
      cidrs = [ var.vpc.cidr_block ]
      port  = 3888
    },
  ]
  out_cidrs = [ var.vpc.cidr_block ]
}

module "zookeeper_configure" {
  source = "../../../shared/kafka/zookeeper"
  count = var.zookeeper_count

  hostname = module.zookeeper_vm.vms[count.index].hostname
  bastion_ip = var.bastion_ip
  ssh_key_path = var.ssh_key_path
  zookeeper_index = count.index
  zookeeper_ips = module.zookeeper_vm.vms[*].ip
}
#-------------------------------------------------------------------------------
# Zookeepers (END)
#-------------------------------------------------------------------------------


#-------------------------------------------------------------------------------
# Brokers
#-------------------------------------------------------------------------------
locals {
  ingress_cidrs = var.public ? [ var.vpc.cidr_block, "${chomp(data.http.myip.body)}/32"] : [ var.vpc.cidr_block ]
  egress_cidrs = var.public ? [ "0.0.0.0/0" ] : [ var.vpc.cidr_block ]
}

module "kafka_vm" {
  source = "../vm"

  name = "kafka"
  stack = var.stack
  cluster = {
    name = var.cluster
    aggregate = false
  }
  vpc_id = var.vpc.id
  dns_zone = var.dns_zone

  vms = [for i in range(var.kafka_count) :
    {
      subnet_id = var.subnets[i % length(var.subnets)].id
      ip = cidrhost(var.subnets[i % length(var.subnets)].cidr_block, 101)
    }
  ]

  image_id = data.aws_ami.kafka.id
  instance_type = "m4.large"
  key_name = aws_key_pair.kafka.key_name
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
      name  = "kafka_internal"
      cidrs = local.ingress_cidrs
      port  = 9092
    },
    {
      name  = "kafka_external"
      cidrs = local.ingress_cidrs
      port  = 9093
    },
  ]
  out_cidrs = local.egress_cidrs
}

module "kafka_configure" {
  source = "../../../shared/kafka"
  count = var.kafka_count

  hostname = module.kafka_vm.vms[count.index].hostname
  bastion_ip = var.bastion_ip
  ssh_key_path = var.ssh_key_path
  public = var.public

  kafka_index = count.index
  kafka_rack = var.azs[count.index % length(var.azs)]
  kafka_internal_ips = module.kafka_vm.vms[*].ip
  kafka_internal_host = module.kafka_vm.vms[count.index].hostname
  kafka_external_host = module.kafka_vm.vms[count.index].public_hostname

  zookeeper_ips = module.zookeeper_vm.vms[*].ip
}
#-------------------------------------------------------------------------------
# Brokers (END)
#-------------------------------------------------------------------------------
