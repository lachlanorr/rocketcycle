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
}

variable "dns_zone" {
  type = any
}

variable "vpc" {
  type = any
}

variable "subnet_app" {
  type = any
}

variable "bastion_ips" {
  type = list
}

variable "balancer_external_urls" {
  type = any
}

variable "balancer_internal_urls" {
  type = any
}

variable "jobs" {
  type = list
}

variable "ssh_key_path" {
  type = string
  default = "~/.ssh/rkcy_id_rsa"
}

variable "prometheus_count" {
  type = number
  default = 1
}
variable "prometheus_port" {
  type = number
  default = 9090
}

variable "grafana_count" {
  type = number
  default = 1
}
variable "grafana_port" {
  type = number
  default = 3000
}

locals {
  sn_ids   = "${values(zipmap(var.subnet_app.*.cidr_block, var.subnet_app.*.id))}"
  sn_cidrs = "${values(zipmap(var.subnet_app.*.cidr_block, var.subnet_app.*.cidr_block))}"
}

data "aws_ami" "metrics" {
  most_recent      = true
  name_regex       = "^rkcy/metrics-[0-9]{8}-[0-9]{6}$"
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
  vpc_id      = var.vpc.id

  ingress = [
    {
      cidr_blocks      = [ var.vpc.cidr_block ]
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
      cidr_blocks      = [ var.vpc.cidr_block ]
      description      = "prometheus server"
      from_port        = var.prometheus_port
      to_port          = var.prometheus_port
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
    },
    {
      cidr_blocks      = [ var.vpc.cidr_block ]
      description      = "node_exporter listener"
      from_port        = 9100
      to_port          = 9100
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
    },
  ]

  egress = [
    {
      cidr_blocks      = [ var.vpc.cidr_block ]
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
  zone_id = var.dns_zone.zone_id
  name    = "prometheus-${count.index}.${var.stack}.local.${var.dns_zone.name}"
  type    = "A"
  ttl     = "300"
  records = [local.prometheus_ips[count.index]]
}

resource "null_resource" "prometheus_provisioner" {
  count = var.prometheus_count
  depends_on = [
    aws_instance.prometheus
  ]

  #---------------------------------------------------------
  # node_exporter
  #---------------------------------------------------------
  provisioner "file" {
    content = templatefile("${path.module}/../../../shared/node_exporter_install.sh", {})
    destination = "/home/ubuntu/node_exporter_install.sh"
  }
  provisioner "remote-exec" {
    inline = [
      <<EOF
sudo bash /home/ubuntu/node_exporter_install.sh
rm /home/ubuntu/node_exporter_install.sh
EOF
    ]
  }
  #---------------------------------------------------------
  # node_exporter (END)
  #---------------------------------------------------------

  provisioner "file" {
    content = templatefile("${path.module}/../../../shared/metrics/prometheus.service.tpl", {
      balancer_url = var.balancer_external_urls.edge
      prometheus_port = var.prometheus_port
    })
    destination = "/home/ubuntu/prometheus.service"
  }

  provisioner "file" {
    content = templatefile("${path.module}/../../../shared/metrics/prometheus.yml.tpl", {
      jobs = concat(var.jobs, [
        {
          name = "prometheus",
          targets = [for host in sort(aws_route53_record.prometheus_private.*.name): "${host}:9100"]
          relabel = [
            {
              source_labels = ["__address__"]
              regex: "([^\\.]+).*"
              target_label = "instance"
              replacement = "$${1}"
            },
          ]
        },
        {
          name = "grafana",
          targets = [for host in sort(aws_route53_record.grafana_private.*.name): "${host}:9100"]
          relabel = [
            {
              source_labels = ["__address__"]
              regex: "([^\\.]+).*"
              target_label = "instance"
              replacement = "$${1}"
            },
          ]
        },
      ])
    })
    destination = "/home/ubuntu/prometheus.yml"
  }

  provisioner "remote-exec" {
    inline = [
      <<EOF
sudo mv /home/ubuntu/prometheus.service /etc/systemd/system/prometheus.service
sudo mv /home/ubuntu/prometheus.yml /opt/prometheus/prometheus.yml
sudo systemctl daemon-reload
sudo systemctl start prometheus
sudo systemctl enable prometheus
EOF
    ]
  }

  connection {
    type     = "ssh"

    bastion_user        = "ubuntu"
    bastion_host        = var.bastion_ips[0]
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
  vpc_id      = var.vpc.id

  ingress = [
    {
      cidr_blocks      = [ var.vpc.cidr_block ]
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
      cidr_blocks      = [ var.vpc.cidr_block ]
      description      = "grafana server"
      from_port        = var.grafana_port
      to_port          = var.grafana_port
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
    },
    {
      cidr_blocks      = [ var.vpc.cidr_block ]
      description      = "node_exporter listener"
      from_port        = 9100
      to_port          = 9100
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
    },
  ]

  egress = [
    {
      cidr_blocks      = [ var.vpc.cidr_block ]
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
  zone_id = var.dns_zone.zone_id
  name    = "grafana-${count.index}.${var.stack}.local.${var.dns_zone.name}"
  type    = "A"
  ttl     = "300"
  records = [local.grafana_ips[count.index]]
}

resource "null_resource" "grafana_provisioner" {
  count = var.grafana_count
  depends_on = [
    aws_instance.grafana
  ]

  #---------------------------------------------------------
  # node_exporter
  #---------------------------------------------------------
  provisioner "file" {
    content = templatefile("${path.module}/../../../shared/node_exporter_install.sh", {})
    destination = "/home/ubuntu/node_exporter_install.sh"
  }
  provisioner "remote-exec" {
    inline = [
      <<EOF
sudo bash /home/ubuntu/node_exporter_install.sh
rm /home/ubuntu/node_exporter_install.sh
EOF
    ]
  }
  #---------------------------------------------------------
  # node_exporter (END)
  #---------------------------------------------------------
  provisioner "file" {
    content = templatefile("${path.module}/../../../shared/metrics/grafana.ini.tpl", {
      balancer_url = var.balancer_external_urls.edge
      grafana_port = var.grafana_port
    })
    destination = "/home/ubuntu/grafana.ini"
  }

  provisioner "file" {
    content = templatefile("${path.module}/../../../shared/metrics/datasources.yaml.tpl", {
      balancer_url = var.balancer_internal_urls.app
    })
    destination = "/home/ubuntu/datasources.yaml"
  }

  provisioner "file" {
    content = templatefile("${path.module}/../../../shared/metrics/dashboards.yaml.tpl", {})
    destination = "/home/ubuntu/dashboards.yaml"
  }

  provisioner "file" {
    source = "${path.module}/../../../shared/metrics/dashboards"
    destination = "/home/ubuntu/dashboards"
  }

  provisioner "remote-exec" {
    inline = [
      <<EOF
sudo mv /etc/grafana/grafana.ini /etc/grafana/grafana.ini.orig
sudo mv /home/ubuntu/grafana.ini /etc/grafana/grafana.ini
sudo chown root:grafana /etc/grafana/grafana.ini

sudo mv /home/ubuntu/datasources.yaml /etc/grafana/provisioning/datasources/datasources.yaml
sudo chown root:grafana /etc/grafana/provisioning/datasources/datasources.yaml

sudo mv /home/ubuntu/dashboards.yaml /etc/grafana/provisioning/dashboards/dashboards.yaml
sudo mv /home/ubuntu/dashboards /etc/grafana/provisioning/dashboards/dashboards
sudo chown -R root:grafana /etc/grafana/provisioning/dashboards

sudo systemctl daemon-reload
sudo systemctl enable grafana-server
sudo systemctl start grafana-server
EOF
    ]
  }

  connection {
    type     = "ssh"

    bastion_user        = "ubuntu"
    bastion_host        = var.bastion_ips[0]
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
  value = sort(aws_route53_record.prometheus_private.*.name)
}
output "prometheus_port" {
  value = var.prometheus_port
}

output "grafana_hosts" {
  value = sort(aws_route53_record.grafana_private.*.name)
}
output "grafana_port" {
  value = var.grafana_port
}
