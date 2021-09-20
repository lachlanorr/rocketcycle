terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
      version = "~> 3.27"
    }
  }

  required_version = ">= 0.14.9"
}

provider "aws" {
  profile = "default"
  region = "us-east-2"
}

variable "stack" {
  type = string
  default = "perfa"
}

variable "balancer_url" {
  type = string
  #default = "http://edge.${var.stack}.${data.aws_route53_zone.zone}"
  default = "http://edge.perfa.rkcy.net"
}

variable "ssh_key_path" {
  type = string
  default = "~/.ssh/rkcy_id_rsa"
}

data "aws_vpc" "rkcy" {
  filter {
    name = "tag:Name"
    values = ["rkcy_${var.stack}_vpc"]
  }
}

variable "bastion_hosts" {
  type = list
  default = ["bastion-0.perfa.rkcy.net"]
}

data "aws_route53_zone" "zone" {
  name = "rkcy.net"
}

data "aws_subnet" "rkcy_app" {
  count = max(var.prometheus_count, var.grafana_count)
  vpc_id = data.aws_vpc.rkcy.id

  filter {
    name = "tag:Name"
    values = ["rkcy_${var.stack}_app_${count.index}_sn"]
  }
}

variable "prometheus_count" {
  type = number
  default = 1
}

variable "grafana_count" {
  type = number
  default = 1
}

locals {
  sn_ids   = "${values(zipmap(data.aws_subnet.rkcy_app.*.cidr_block, data.aws_subnet.rkcy_app.*.id))}"
  sn_cidrs = "${values(zipmap(data.aws_subnet.rkcy_app.*.cidr_block, data.aws_subnet.rkcy_app.*.cidr_block))}"
}

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
locals {
  prometheus_ips = [for i in range(var.prometheus_count) : "${cidrhost(local.sn_cidrs[i], 90)}"]
}

resource "aws_security_group" "rkcy_prometheus" {
  name        = "rkcy_${var.stack}_prometheus"
  description = "Allow SSH and prometheus inbound traffic"
  vpc_id      = data.aws_vpc.rkcy.id

  ingress = [
    {
      cidr_blocks      = [ data.aws_vpc.rkcy.cidr_block ]
      description      = ""
      from_port        = 22
      to_port          = 22
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
    },
    {
      cidr_blocks      = [ data.aws_vpc.rkcy.cidr_block ]
      description      = "prometheus server"
      from_port        = 9090
      to_port          = 9090
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
    },
  ]

  egress = [
    {
      cidr_blocks      = [ data.aws_vpc.rkcy.cidr_block ]
      description      = ""
      from_port        = 0
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "-1"
      security_groups  = []
      self             = false
      to_port          = 0
    }
  ]
}

resource "aws_network_interface" "prometheus" {
  count = var.prometheus_count
  subnet_id   = local.sn_ids[count.index]
  private_ips = [local.prometheus_ips[count.index]]

  security_groups = [aws_security_group.rkcy_prometheus.id]
}

resource "aws_placement_group" "prometheus" {
  name     = "rkcy_${var.stack}_prometheus_pc"
  strategy = "spread"
}

resource "aws_instance" "prometheus" {
  count = var.prometheus_count
  ami = data.aws_ami.metrics.id
  instance_type = "m4.large"
  placement_group = aws_placement_group.prometheus.name

  key_name = aws_key_pair.metrics.key_name

  network_interface {
    network_interface_id = aws_network_interface.prometheus[count.index].id
    device_index = 0
  }

  credit_specification {
    cpu_credits = "unlimited"
  }

  tags = {
    Name = "rkcy_${var.stack}_inst_prometheus_${count.index}"
  }
}

resource "aws_route53_record" "prometheus_private" {
  count = var.prometheus_count
  zone_id = data.aws_route53_zone.zone.zone_id
  name    = "prometheus-${count.index}.${var.stack}.local.${data.aws_route53_zone.zone.name}"
  type    = "A"
  ttl     = "300"
  records = [local.prometheus_ips[count.index]]


  provisioner "file" {
    content = templatefile("${path.module}/prometheus.service.tpl", {
      balancer_url = var.balancer_url
    })
    destination = "/home/ubuntu/prometheus.service"
  }

  provisioner "remote-exec" {
    inline = [
      <<EOF
sudo mv /home/ubuntu/prometheus.service /etc/systemd/system/prometheus.service
sudo systemctl daemon-reload
sudo systemctl start prometheus
sudo systemctl enable prometheus
EOF
    ]
  }

  connection {
    type     = "ssh"

    bastion_user        = "ubuntu"
    bastion_host        = var.bastion_hosts[0]
    bastion_private_key = file(var.ssh_key_path)

    user        = "ubuntu"
    host        = local.prometheus_ips[count.index]
    private_key = file(var.ssh_key_path)
  }
}
#-------------------------------------------------------------------------------
# prometheus (END)
#-------------------------------------------------------------------------------

#-------------------------------------------------------------------------------
# grafana
#-------------------------------------------------------------------------------
locals {
  grafana_ips = [for i in range(var.grafana_count) : "${cidrhost(local.sn_cidrs[i], 30)}"]
}

resource "aws_security_group" "rkcy_grafana" {
  name        = "rkcy_${var.stack}_grafana"
  description = "Allow SSH and grafana inbound traffic"
  vpc_id      = data.aws_vpc.rkcy.id

  ingress = [
    {
      cidr_blocks      = [ data.aws_vpc.rkcy.cidr_block ]
      description      = ""
      from_port        = 22
      to_port          = 22
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
    },
    {
      cidr_blocks      = [ data.aws_vpc.rkcy.cidr_block ]
      description      = "grafana server"
      from_port        = 3000
      to_port          = 3000
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
    },
  ]

  egress = [
    {
      cidr_blocks      = [ data.aws_vpc.rkcy.cidr_block ]
      description      = ""
      from_port        = 0
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "-1"
      security_groups  = []
      self             = false
      to_port          = 0
    }
  ]
}

resource "aws_network_interface" "grafana" {
  count = var.grafana_count
  subnet_id   = local.sn_ids[count.index]
  private_ips = [local.grafana_ips[count.index]]

  security_groups = [aws_security_group.rkcy_grafana.id]
}

resource "aws_placement_group" "grafana" {
  name     = "rkcy_${var.stack}_grafana_pc"
  strategy = "spread"
}

resource "aws_instance" "grafana" {
  count = var.grafana_count
  ami = data.aws_ami.metrics.id
  instance_type = "m4.large"
  placement_group = aws_placement_group.grafana.name

  key_name = aws_key_pair.metrics.key_name

  network_interface {
    network_interface_id = aws_network_interface.grafana[count.index].id
    device_index = 0
  }

  credit_specification {
    cpu_credits = "unlimited"
  }

  tags = {
    Name = "rkcy_${var.stack}_inst_grafana_${count.index}"
  }
}

resource "aws_route53_record" "grafana_private" {
  count = var.grafana_count
  zone_id = data.aws_route53_zone.zone.zone_id
  name    = "grafana-${count.index}.${var.stack}.local.${data.aws_route53_zone.zone.name}"
  type    = "A"
  ttl     = "300"
  records = [local.grafana_ips[count.index]]

  provisioner "file" {
    content = templatefile("${path.module}/grafana.ini.tpl", {
      balancer_url = var.balancer_url
    })
    destination = "/home/ubuntu/grafana.ini"
  }

  provisioner "remote-exec" {
    inline = [
      <<EOF
sudo mv /etc/grafana/grafana.ini /etc/grafana/grafana.ini.orig
sudo mv /home/ubuntu/grafana.ini /etc/grafana/grafana.ini
sudo chown root:grafana /etc/grafana/grafana.ini

sudo systemctl daemon-reload
sudo systemctl enable grafana-server
sudo systemctl start grafana-server
EOF
    ]
  }

  connection {
    type     = "ssh"

    bastion_user        = "ubuntu"
    bastion_host        = var.bastion_hosts[0]
    bastion_private_key = file(var.ssh_key_path)

    user        = "ubuntu"
    host        = local.grafana_ips[count.index]
    private_key = file(var.ssh_key_path)
  }
}
#-------------------------------------------------------------------------------
# grafana (END)
#-------------------------------------------------------------------------------

output "prometheus_hosts" {
  value = [for host in aws_route53_record.prometheus_private.*.name: "${host}:9090"]
}
