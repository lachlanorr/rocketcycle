data "azurerm_image" "dev" {
  name_regex          = "^rkcy-dev-[0-9]{8}-[0-9]{6}$"
  sort_descending     = true
  resource_group_name = var.image_resource_group_name
}

data "http" "myip" {
  url = "http://ipv4.icanhazip.com"
}

module "dev_vm" {
  source = "../vm"

  name = "dev"
  stack = var.stack
  dns_resource_group_name = var.image_resource_group_name
  ssh_key_path = var.ssh_key_path
  resource_group = var.resource_group
  network_cidr = var.network_cidr
  dns_zone = var.dns_zone
  azs = var.azs

  vms = [
    {
      subnet_id = var.subnets[0].id
      ip = cidrhost(var.subnets[0].address_prefixes[0], 113)
    }
  ]

  image_id = data.azurerm_image.dev.id
  size = "Standard_D2d_v4"
  public = true

  in_rules = [
    {
      name  = "ssh"
      cidrs = [ var.network_cidr, "${chomp(data.http.myip.body)}/32" ]
      port  = 22
    },
    {
      name  = "node_exporter"
      cidrs = [ var.network_cidr, "${chomp(data.http.myip.body)}/32" ]
      port  = 9100
    },
  ]
}

module "dev_configure" {
  source = "../../../shared/dev"

  hostname = module.dev_vm.vms[0].hostname
  dev_public_ip = module.dev_vm.vms[0].public_ip
  ssh_key_path = var.ssh_key_path
  postgresql_hosts = var.postgresql_hosts
  kafka_hosts = var.kafka_hosts
  kafka_cluster = var.kafka_cluster
  otelcol_endpoint = var.otelcol_endpoint
}
