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

variable "cluster" {
  type = string
}

variable "vpc" {
  type = any
}

variable "subnet_app" {
  type = any
}

variable "dns_zone" {
  type = any
}

variable "bastion_hosts" {
  type = list
}

variable "ssh_key_path" {
  type = string
  default = "~/.ssh/rkcy_id_rsa"
}

variable "zookeeper_count" {
  type = number
  default = 3
}

variable "kafka_count" {
  type = number
  default = 3
}

locals {
  sn_ids   = "${values(zipmap(var.subnet_app.*.cidr_block, var.subnet_app.*.id))}"
  sn_cidrs = "${values(zipmap(var.subnet_app.*.cidr_block, var.subnet_app.*.cidr_block))}"
}

data "aws_ami" "kafka" {
  most_recent      = true
  name_regex       = "^rkcy-kafka-[0-9]{8}-[0-9]{6}$"
  owners           = ["self"]
}

resource "aws_key_pair" "kafka" {
  key_name = "rkcy-${var.cluster}-${var.stack}-kafka"
  public_key = file("${var.ssh_key_path}.pub")
}

#-------------------------------------------------------------------------------
# Zookeepers
#-------------------------------------------------------------------------------
locals {
  zookeeper_ips = [for i in range(var.zookeeper_count) : "${cidrhost(local.sn_cidrs[i], 100)}"]
}

resource "aws_security_group" "rkcy_zookeeper" {
  name        = "rkcy_${var.cluster}_${var.stack}_zookeeper"
  description = "Allow SSH and zookeeper inbound traffic"
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
      description      = ""
      from_port        = 2181
      to_port          = 2181
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
    },
    {
      cidr_blocks      = [ var.vpc.cidr_block ]
      description      = ""
      from_port        = 2888
      to_port          = 2888
      ipv6_cidr_blocks = []
      prefix_list_ids  = []
      protocol         = "tcp"
      security_groups  = []
      self             = false
    },
    {
      cidr_blocks      = [ var.vpc.cidr_block ]
      description      = ""
      from_port        = 3888
      to_port          = 3888
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

resource "aws_network_interface" "zookeeper" {
  count = var.zookeeper_count
  subnet_id   = local.sn_ids[count.index]
  private_ips = [local.zookeeper_ips[count.index]]

  security_groups = [aws_security_group.rkcy_zookeeper.id]
}

resource "aws_placement_group" "zookeeper" {
  name     = "rkcy_${var.cluster}_${var.stack}_zookeeper_pc"
  strategy = "spread"
}

resource "aws_instance" "zookeeper" {
  count = var.zookeeper_count
  ami = data.aws_ami.kafka.id
  instance_type = "m4.large"
  placement_group = aws_placement_group.zookeeper.name

  key_name = aws_key_pair.kafka.key_name

  network_interface {
    network_interface_id = aws_network_interface.zookeeper[count.index].id
    device_index = 0
  }

  credit_specification {
    cpu_credits = "unlimited"
  }

  tags = {
    Name = "rkcy_${var.cluster}_${var.stack}_inst_zookeeper_${count.index+1}"
  }
}

resource "aws_route53_record" "zookeeper_private" {
  count = var.zookeeper_count
  zone_id = var.dns_zone.zone_id
  name    = "zookeeper-${count.index+1}.${var.cluster}.${var.stack}.local.${var.dns_zone.name}"
  type    = "A"
  ttl     = "300"
  records = [local.zookeeper_ips[count.index]]


  provisioner "file" {
    content = templatefile("${path.module}/zookeeper.service.tpl", {})
    destination = "/home/ubuntu/zookeeper.service"
  }

  provisioner "file" {
    content = templatefile(
      "${path.module}/zookeeper.properties.tpl",
      {
        zookeeper_ips = local.zookeeper_ips
      })
    destination = "/home/ubuntu/zookeeper.properties"
  }

  provisioner "remote-exec" {
    inline = [
      <<EOF
sudo mv /home/ubuntu/zookeeper.service /etc/systemd/system/zookeeper.service
sudo mv /home/ubuntu/zookeeper.properties /etc/zookeeper.properties
sudo chown kafka:kafka /etc/zookeeper.properties

sudo mkdir -p /data/zookeeper
sudo bash -c 'echo ${count.index+1} > /data/zookeeper/myid'
sudo chown -R kafka:kafka /data
sudo systemctl daemon-reload
sudo systemctl start zookeeper
sudo systemctl enable zookeeper
EOF
    ]
  }

  connection {
    type     = "ssh"

    bastion_user        = "ubuntu"
    bastion_host        = var.bastion_hosts[0]
    bastion_private_key = file(var.ssh_key_path)

    user        = "ubuntu"
    host        = local.zookeeper_ips[count.index]
    private_key = file(var.ssh_key_path)
  }
}
#-------------------------------------------------------------------------------
# Zookeepers (END)
#-------------------------------------------------------------------------------


#-------------------------------------------------------------------------------
# Brokers
#-------------------------------------------------------------------------------
locals {
  kafka_ips = [for i in range(var.kafka_count) : "${cidrhost(local.sn_cidrs[i], 101)}"]
  kafka_hosts = [for i in range(var.kafka_count) : "kafka-${i}.${var.cluster}.${var.stack}.local.${var.dns_zone.name}"]
}

resource "aws_security_group" "rkcy_kafka" {
  name        = "rkcy_${var.cluster}_${var.stack}_kafka"
  description = "Allow SSH and kafka inbound traffic"
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
      description      = ""
      from_port        = 9092
      to_port          = 9092
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

resource "aws_network_interface" "kafka" {
  count = var.kafka_count
  subnet_id   = local.sn_ids[count.index]
  private_ips = [local.kafka_ips[count.index]]

  security_groups = [aws_security_group.rkcy_kafka.id]
}

resource "aws_placement_group" "kafka" {
  name     = "rkcy_${var.cluster}_${var.stack}_kafka_pc"
  strategy = "spread"
}

resource "aws_instance" "kafka" {
  count = var.kafka_count
  ami = data.aws_ami.kafka.id
  instance_type = "m4.large"
  placement_group = aws_placement_group.kafka.name

  key_name = aws_key_pair.kafka.key_name

  network_interface {
    network_interface_id = aws_network_interface.kafka[count.index].id
    device_index = 0
  }

  credit_specification {
    cpu_credits = "unlimited"
  }

  tags = {
    Name = "rkcy_${var.cluster}_${var.stack}_inst_kafka_${count.index}"
  }
}

resource "aws_route53_record" "kafka_private" {
  count = var.kafka_count
  zone_id = var.dns_zone.zone_id
  name    = local.kafka_hosts[count.index]
  type    = "A"
  ttl     = "300"
  records = [local.kafka_ips[count.index]]


  provisioner "file" {
    content = templatefile("${path.module}/kafka.service.tpl", {})
    destination = "/home/ubuntu/kafka.service"
  }

  provisioner "file" {
    content = templatefile(
      "${path.module}/kafka.properties.tpl",
      {
        idx = count.index,
        kafka_ips = local.kafka_ips,
        kafka_hosts = local.kafka_hosts,
        zookeeper_ips = local.zookeeper_ips
      })
    destination = "/home/ubuntu/kafka.properties"
  }

  provisioner "remote-exec" {
    inline = [
      <<EOF
sudo mv /home/ubuntu/kafka.service /etc/systemd/system/kafka.service
sudo mv /home/ubuntu/kafka.properties /etc/kafka.properties
sudo chown kafka:kafka /etc/kafka.properties

sudo mkdir -p /data/kafka
sudo chown -R kafka:kafka /data
sudo systemctl daemon-reload

%{for ip in local.zookeeper_ips}
RET=1
while [ \$RET -ne 0]; do
  echo Trying zookeeper ${ip}:2181
  nc -z ${ip} 2181
  RET=\$?
done
echo Connected zookeeper ${ip}:2181
%{endfor}

sleep 10

sudo systemctl start kafka
sudo systemctl enable kafka
EOF
    ]
  }

  connection {
    type     = "ssh"

    bastion_user        = "ubuntu"
    bastion_host        = var.bastion_hosts[0]
    bastion_private_key = file(var.ssh_key_path)

    user        = "ubuntu"
    host        = local.kafka_ips[count.index]
    private_key = file(var.ssh_key_path)
  }
}

output "kafka_cluster" {
  value = "${var.stack}_${var.cluster}"
}

output "kafka_hosts" {
  value = sort(aws_route53_record.kafka_private.*.name)
}
#-------------------------------------------------------------------------------
# Brokers (END)
#-------------------------------------------------------------------------------
