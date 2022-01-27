data "azurerm_image" "kafka" {
  name_regex          = "^rkcy-kafka-[0-9]{8}-[0-9]{6}$"
  sort_descending     = true
  resource_group_name = var.image_resource_group_name
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
  resource_group = var.resource_group
  network_cidr = var.network_cidr
  dns_resource_group_name = var.image_resource_group_name
  ssh_key_path = var.ssh_key_path
  dns_zone = var.dns_zone
  azs = var.azs

  vms = [for i in range(var.zookeeper_count) :
    {
      subnet_id = var.subnets[i % length(var.subnets)].id
      ip = cidrhost(var.subnets[i % length(var.subnets)].address_prefixes[0], 100)
    }
  ]

  image_id = data.azurerm_image.kafka.id
  size = "Standard_D2d_v4"
  public = false

  in_rules = [
    {
      name  = "ssh"
      cidrs = [ var.network_cidr ]
      port  = 22
    },
    {
      name  = "node_exporter"
      cidrs = [ var.network_cidr ]
      port  = 9100
    },
    {
      name  = "zookeeper_client"
      cidrs = [ var.network_cidr ]
      port  = 2181
    },
    {
      name  = "zookeeper_peer"
      cidrs = [ var.network_cidr ]
      port  = 2888
    },
    {
      name  = "zookeeper_leader"
      cidrs = [ var.network_cidr ]
      port  = 3888
    },
  ]
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
  ingress_cidrs = var.public ? [ var.network_cidr, "${chomp(data.http.myip.body)}/32"] : [ var.network_cidr ]
}

module "kafka_vm" {
  source = "../vm"

  name = "kafka"
  stack = var.stack
  cluster = {
    name = var.cluster
    aggregate = false
  }
  dns_resource_group_name = var.image_resource_group_name
  ssh_key_path = var.ssh_key_path
  resource_group = var.resource_group
  network_cidr = var.network_cidr
  dns_zone = var.dns_zone
  azs = var.azs

  vms = [for i in range(var.kafka_count) :
    {
      subnet_id = var.subnets[i % length(var.subnets)].id
      ip = cidrhost(var.subnets[i % length(var.subnets)].address_prefixes[0], 101)
    }
  ]

  image_id = data.azurerm_image.kafka.id
  size = "Standard_D2d_v4"
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
