variable "stack" {
  type = string
}

variable "cluster" {
  type = string
}

variable "vpc" {
  type = any
}

variable "subnet" {
  type = any
}

variable "dns_zone" {
  type = any
}

variable "bastion_ips" {
  type = list
}

variable "inbound_cidr" {
  type = string
}

variable "routes" {
  type = any
}

variable "ssh_key_path" {
  type = string
  default = "~/.ssh/rkcy_id_rsa"
}

variable "nginx_count" {
  type = number
  default = 1
}

data "aws_ami" "nginx" {
  most_recent      = true
  name_regex       = "^rkcy-nginx-[0-9]{8}-[0-9]{6}$"
  owners           = ["self"]
}

locals {
  sn_ids   = "${values(zipmap(var.subnet.*.cidr_block, var.subnet.*.id))}"
  sn_cidrs = "${values(zipmap(var.subnet.*.cidr_block, var.subnet.*.cidr_block))}"

  nginx_ips = [for i in range(var.nginx_count) : "${cidrhost(local.sn_cidrs[i], 80)}"]
}

variable "public" {
  type = bool
}
data "http" "myip" {
  url = "http://ipv4.icanhazip.com"
}
locals {
  ingress_cidrs = var.public ? [ var.vpc.cidr_block, "${chomp(data.http.myip.body)}/32"] : [ var.vpc.cidr_block ]
  ingress_80443_cidrs = var.public ? [ var.inbound_cidr, "${chomp(data.http.myip.body)}/32"] : [ var.inbound_cidr ]
}

resource "aws_security_group" "rkcy_nginx" {
  name        = "rkcy_${var.cluster}_${var.stack}_nginx"
  description = "Allow SSH, HTTP, and HTTPS inbound traffic"
  vpc_id      = var.vpc.id

  ingress = [
    {
      cidr_blocks      = local.ingress_cidrs
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
      cidr_blocks      = local.ingress_80443_cidrs
      description      = ""
      from_port        = 80
      to_port          = 80
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
    },
    {
      cidr_blocks      = local.ingress_80443_cidrs
      description      = ""
      from_port        = 443
      to_port          = 443
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
    },
    {
      cidr_blocks      = local.ingress_cidrs
      description      = "node_exporter"
      from_port        = 9100
      to_port          = 9100
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
    },
    {
      cidr_blocks      = local.ingress_cidrs
      description      = "otelcol"
      from_port        = 4317
      to_port          = 4317
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
    },
    {
      cidr_blocks      = local.ingress_cidrs
      description      = "jaeger collector"
      from_port        = 14250
      to_port          = 14250
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
    },
  ]

  egress = [
    {
      cidr_blocks      = [ "0.0.0.0/0" ]
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

resource "aws_network_interface" "nginx" {
  count = var.nginx_count
  subnet_id   = local.sn_ids[count.index]
  private_ips = [local.nginx_ips[count.index]]

  security_groups = [aws_security_group.rkcy_nginx.id]
}

resource "aws_placement_group" "nginx" {
  name     = "rkcy_${var.cluster}_${var.stack}_nginx_pc"
  strategy = "spread"
}

resource "aws_key_pair" "nginx" {
  key_name = "rkcy_${var.cluster}_${var.stack}_nginx"
  public_key = file("${var.ssh_key_path}.pub")
}

resource "aws_instance" "nginx" {
  count = var.nginx_count
  ami = data.aws_ami.nginx.id
  instance_type = "m4.large"
  placement_group = aws_placement_group.nginx.name

  key_name = aws_key_pair.nginx.key_name

  network_interface {
    network_interface_id = aws_network_interface.nginx[count.index].id
    device_index = 0
  }

  credit_specification {
    cpu_credits = "unlimited"
  }

  tags = {
    Name = "rkcy_${var.cluster}_${var.stack}_inst_nginx_${count.index}"
  }
}

resource "aws_eip" "nginx" {
  count = var.public ? var.nginx_count : 0
  vpc = true

  instance = aws_instance.nginx[count.index].id
  associate_with_private_ip = local.nginx_ips[count.index]

  tags = {
    Name = "rkcy_${var.cluster}_${var.stack}_eip_nginx_${count.index}"
  }
}

resource "aws_route53_record" "nginx_public" {
  count = var.public ? 1 : 0
  zone_id = var.dns_zone.zone_id
  name    = "${var.cluster}.${var.stack}.${var.dns_zone.name}"
  type    = "A"
  ttl     = "300"
  records = aws_eip.nginx.*.public_ip
}

resource "aws_route53_record" "nginx_private_aggregate" {
  zone_id = var.dns_zone.zone_id
  name    = "${var.cluster}.${var.stack}.local.${var.dns_zone.name}"
  type    = "A"
  ttl     = "300"
  records = local.nginx_ips
}

resource "aws_route53_record" "nginx_private" {
  count = var.nginx_count
  zone_id = var.dns_zone.zone_id
  name    = "nginx-${count.index}.${var.cluster}.${var.stack}.local.${var.dns_zone.name}"
  type    = "A"
  ttl     = "300"
  records = [local.nginx_ips[count.index]]
}

resource "null_resource" "nginx_provisioner" {
  count = var.nginx_count
  depends_on = [
    aws_instance.nginx
  ]

  #---------------------------------------------------------
  # node_exporter
  #---------------------------------------------------------
  provisioner "file" {
    content = templatefile("${path.module}/../../shared/node_exporter_install.sh", {})
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
    content = templatefile("${path.module}/index.html.tpl", {})
    destination = "/home/ubuntu/index.html"
  }

  provisioner "file" {
    content = templatefile("${path.module}/default.tpl", {
      routes = var.routes
    })
    destination = "/home/ubuntu/default"
  }

  provisioner "remote-exec" {
    inline = [
      <<EOF
sudo rm -rf /var/www/html/*
sudo mv /home/ubuntu/index.html /var/www/html/index.html
sudo chown root:root /var/www/html/index.html

sudo mv /home/ubuntu/default /etc/nginx/sites-enabled/default

sudo systemctl reload nginx

EOF
    ]
  }

  connection {
    type     = "ssh"

    bastion_user        = "ubuntu"
    bastion_host        = var.bastion_ips[0]
    bastion_private_key = file(var.ssh_key_path)

    user        = "ubuntu"
    host        = local.nginx_ips[count.index]
    private_key = file(var.ssh_key_path)
  }
}

output "balancer_internal_url" {
  value = "http://${aws_route53_record.nginx_private_aggregate.name}"
}

output "balancer_external_url" {
  value = var.public ? "http://${aws_route53_record.nginx_public[0].name}" : ""
}

output "nginx_hosts" {
  value = sort(aws_route53_record.nginx_private.*.name)
}
