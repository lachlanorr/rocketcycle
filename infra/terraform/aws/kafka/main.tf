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

variable "ssh_key_path" {
  type = string
  default = "~/.ssh/rkcy_id_rsa"
}

variable "zookeeper_count" {
  type = number
  default = 3
}

variable "zookeeper_data_dir" {
  type = string
  default = "/data/zookeeper"
}

variable "zookeeper_properties_file" {
  type = string
  default = "/etc/zookeeper.properties"
}

data "aws_vpc" "rkcy" {
  filter {
    name = "tag:Name"
    values = ["rkcy_vpc"]
  }
}

data "aws_subnet" "rkcy" {
  vpc_id = data.aws_vpc.rkcy.id

  filter {
    name = "tag:Name"
    values = ["rkcy_sn"]
  }
}

resource "aws_security_group" "rkcy_zookeeper" {
  name        = "rkcy_zookeeper"
  description = "Allow SSH and zookeeper inbound traffic"
  vpc_id      = data.aws_vpc.rkcy.id

  ingress = [
    {
      cidr_blocks      = [ "0.0.0.0/0", ]
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
      cidr_blocks      = [ "0.0.0.0/0", ]
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
      cidr_blocks      = [ "0.0.0.0/0", ]
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
      cidr_blocks      = [ "0.0.0.0/0", ]
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
      cidr_blocks      = [ "0.0.0.0/0", ]
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

locals {
  zookeeper_ips = [for i in range(var.zookeeper_count) : "${cidrhost(data.aws_subnet.rkcy.cidr_block, 21 + i)}"]
}

resource "aws_network_interface" "zookeeper" {
  count = var.zookeeper_count
  subnet_id   = data.aws_subnet.rkcy.id
  private_ips = [local.zookeeper_ips[count.index]]

  security_groups = [aws_security_group.rkcy_zookeeper.id]
}

data "aws_ami" "kafka" {
  most_recent      = true
  name_regex       = "^rkcy-kafka-[0-9]{8}-[0-9]{6}$"
  owners           = ["self"]
}

resource "aws_key_pair" "kafka" {
  key_name = "rkcy-kafka"
  public_key = file("${var.ssh_key_path}.pub")
}

resource "aws_instance" "zookeeper" {
  count = var.zookeeper_count
  ami = data.aws_ami.kafka.id
  instance_type = "m4.large"

  key_name = aws_key_pair.kafka.key_name

  network_interface {
    network_interface_id = aws_network_interface.zookeeper[count.index].id
    device_index = 0
  }

  credit_specification {
    cpu_credits = "unlimited"
  }

  provisioner "remote-exec" {
    inline = [
      "sudo mkdir -p ${var.zookeeper_data_dir}"
      , "sudo chown -R kafka:kafka ${var.zookeeper_data_dir}"

      , <<EOF
sudo bash -c 'cat > /etc/systemd/system/zookeeper.service << EOF
[Unit]
Description=Apache Zookeeper server (Kafka)
Documentation=http://zookeeper.apache.org
Requires=network.target remote-fs.target
After=network.target remote-fs.target

[Service]
Type=simple
User=kafka
Group=kafka
Environment=JAVA_HOME=/usr/lib/jvm/java-11-openjdk-amd64
ExecStart=/opt/kafka/bin/zookeeper-server-start.sh ${var.zookeeper_properties_file}
ExecStop=/opt/kafka/bin/zookeeper-server-stop.sh

[Install]
WantedBy=multi-user.target
EOF'
EOF
      , <<EOF
sudo bash -c 'cat > ${var.zookeeper_properties_file} << EOF
tickTime=2000
dataDir=${var.zookeeper_data_dir}
clientPort=2181
initLimit=5
syncLimit=2
4lw.commands.whitelist=*

%{ for i, host in local.zookeeper_ips ~}
server.${i + 1}=${host}:2888:3888
%{ endfor ~}
EOF'
EOF
      , "sudo bash -c 'echo \"${count.index+1}\" > ${var.zookeeper_data_dir}/myid'"
      , "sudo touch ${var.zookeeper_data_dir}/initialize"
      , "sudo chown kafka:kafka ${var.zookeeper_properties_file} ${var.zookeeper_data_dir}/myid ${var.zookeeper_data_dir}/initialize"
      , "sudo systemctl daemon-reload"
      , "sudo systemctl start zookeeper"
      , "sudo systemctl enable zookeeper"
    ]
  }

  connection {
    type     = "ssh"

    bastion_user        = "ubuntu"
    bastion_host        = "bastion.${data.aws_route53_zone.rkcy_net.name}"
    bastion_private_key = file(var.ssh_key_path)

    user        = "ubuntu"
    host        = local.zookeeper_ips[count.index]
    private_key = file(var.ssh_key_path)
  }

  tags = {
    Name = "rkcy_inst_zookeeper_${count.index+1}"
  }
}

data "aws_route53_zone" "rkcy_net" {
  name = "rkcy.net"
}

resource "aws_route53_record" "zookeeper_private" {
  count = var.zookeeper_count
  zone_id = data.aws_route53_zone.rkcy_net.zone_id
  name    = "zk${count.index+1}.local.${data.aws_route53_zone.rkcy_net.name}"
  type    = "A"
  ttl     = "300"
  records = [local.zookeeper_ips[count.index]]
}



#sudo bash -c 'cat > /etc/systemd/system/kafka.service << EOF
#[Unit]
#Description=Apache Kafka server (broker)
#Documentation=http://kafka.apache.org/documentation.html
#Requires=network.target remote-fs.target
#After=network.target remote-fs.target zookeeper.service
#
#[Service]
#Type=simple
#User=kafka
#Group=kafka
#Environment=JAVA_HOME=/usr/lib/jvm/java-11-openjdk-amd64
#ExecStart=/opt/kafka/bin/kafka-server-start.sh /etc/kafka.properties
#ExecStop=/opt/kafka/bin/kafka-server-stop.sh
#
#[Install]
#WantedBy=multi-user.target
#EOF'

