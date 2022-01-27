data "azurerm_image" "elasticsearch" {
  name_regex          = "^rkcy-elasticsearch-[0-9]{8}-[0-9]{6}$"
  sort_descending     = true
  resource_group_name = var.image_resource_group_name
}

module "elasticsearch_vm" {
  source = "../vm"

  name = "elasticsearch"
  stack = var.stack
  dns_resource_group_name = var.image_resource_group_name
  ssh_key_path = var.ssh_key_path
  resource_group = var.resource_group
  network_cidr = var.network_cidr
  dns_zone = var.dns_zone
  azs = var.azs

  vms = [for i in range(var.elasticsearch_count) :
    {
      subnet_id = var.subnets[i % length(var.subnets)].id
      ip = cidrhost(var.subnets[i % length(var.subnets)].address_prefixes[0], 92)
    }
  ]

  image_id = data.azurerm_image.elasticsearch.id
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
      name  = "elasticsearch_rest"
      cidrs = [ var.network_cidr ]
      port  = var.elasticsearch_port
    },
    {
      name  = "elasticsearch_nodes"
      cidrs = [ var.network_cidr ]
      port  = 9300
    },
  ]
}

module "elasticsearch_configure" {
  source = "../../../shared/elasticsearch"
  count = var.elasticsearch_count

  hostname = module.elasticsearch_vm.vms[count.index].hostname
  bastion_ip = var.bastion_ip
  ssh_key_path = var.ssh_key_path
  stack = var.stack
  elasticsearch_index = count.index
  elasticsearch_ips = module.elasticsearch_vm.vms[*].ip
  elasticsearch_nodes = [for i in range(var.elasticsearch_count) : "master-${i}"]
  elasticsearch_rack = var.azs[count.index % length(var.azs)]
}
