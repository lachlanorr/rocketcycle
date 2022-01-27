data "aws_ami" "elasticsearch" {
  most_recent      = true
  name_regex       = "^rkcy-elasticsearch-[0-9]{8}-[0-9]{6}$"
  owners           = ["self"]
}

resource "aws_key_pair" "elasticsearch" {
  key_name = "rkcy-${var.stack}-elasticsearch"
  public_key = file("${var.ssh_key_path}.pub")
}

module "elasticsearch_vm" {
  source = "../vm"

  name = "elasticsearch"
  stack = var.stack
  vpc_id = var.vpc.id
  dns_zone = var.dns_zone

  vms = [for i in range(var.elasticsearch_count) :
    {
      subnet_id = var.subnets[i % length(var.subnets)].id
      ip = cidrhost(var.subnets[i % length(var.subnets)].cidr_block, 92)
    }
  ]

  image_id = data.aws_ami.elasticsearch.id
  instance_type = "m4.large"
  key_name = aws_key_pair.elasticsearch.key_name
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
      name  = "elasticsearch_rest"
      cidrs = [ var.vpc.cidr_block ]
      port  = var.elasticsearch_port
    },
    {
      name  = "elasticsearch_nodes"
      cidrs = [ var.vpc.cidr_block ]
      port  = 9300
    },
  ]
  out_cidrs = [ var.vpc.cidr_block ]
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
