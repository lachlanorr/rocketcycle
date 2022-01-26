data "aws_ami" "dev" {
  most_recent      = true
  name_regex       = "^rkcy-dev-[0-9]{8}-[0-9]{6}$"
  owners           = ["self"]
}

resource "aws_key_pair" "dev" {
  key_name = "rkcy_${var.stack}_dev"
  public_key = file("${var.ssh_key_path}.pub")
}

data "http" "myip" {
  url = "http://ipv4.icanhazip.com"
}

module "dev_vm" {
  source = "../vm"

  name = "dev"
  stack = var.stack
  vpc_id = var.vpc.id
  dns_zone = var.dns_zone

  vms = [
    {
      subnet_id = var.subnets[0].id
      ip = cidrhost(var.subnets[0].cidr_block, 113)
    }
  ]

  image_id = data.aws_ami.dev.id
  instance_type = "m4.large"
  key_name = aws_key_pair.dev.key_name
  public = true

  in_rules = [
    {
      name  = "ssh"
      cidrs = [ var.vpc.cidr_block, "${chomp(data.http.myip.body)}/32" ]
      port  = 22
    },
    {
      name  = "node_exporter"
      cidrs = [ var.vpc.cidr_block, "${chomp(data.http.myip.body)}/32" ]
      port  = 9100
    },
  ]
  out_cidrs = [ "0.0.0.0/0" ]
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
