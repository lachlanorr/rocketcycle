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

variable "kafka_count" {
  type = number
  default = 3
}

variable "kafka_data_dir" {
  type = string
  default = "/data/kafka"
}

variable "kafka_properties_file" {
  type = string
  default = "/etc/kafka.properties"
}

data "aws_vpc" "rkcy" {
  filter {
    name = "tag:Name"
    values = ["rkcy_vpc"]
  }
}

data "aws_subnet" "rkcy_app" {
  count = max(var.zookeeper_count, var.kafka_count)
  vpc_id = data.aws_vpc.rkcy.id

  filter {
    name = "tag:Name"
    values = ["rkcy_app${count.index+1}_sn"]
  }
}

locals {
  sn_ids   = "${values(zipmap(data.aws_subnet.rkcy_app.*.availability_zone, data.aws_subnet.rkcy_app.*.id))}"
  sn_cidrs = "${values(zipmap(data.aws_subnet.rkcy_app.*.availability_zone, data.aws_subnet.rkcy_app.*.cidr_block))}"
}

data "aws_ami" "kafka" {
  most_recent      = true
  name_regex       = "^rkcy-kafka-[0-9]{8}-[0-9]{6}$"
  owners           = ["self"]
}

data "aws_route53_zone" "rkcy_net" {
  name = "rkcy.net"
}

resource "aws_key_pair" "kafka" {
  key_name = "rkcy-kafka"
  public_key = file("${var.ssh_key_path}.pub")
}

#-------------------------------------------------------------------------------
# Zookeepers
#-------------------------------------------------------------------------------
locals {
  zookeeper_ips = [for i in range(var.zookeeper_count) : "${cidrhost(local.sn_cidrs[i], 100)}"]
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

resource "aws_network_interface" "zookeeper" {
  count = var.zookeeper_count
  subnet_id   = local.sn_ids[count.index]
  private_ips = [local.zookeeper_ips[count.index]]

  security_groups = [aws_security_group.rkcy_zookeeper.id]
}

resource "aws_placement_group" "zookeeper" {
  name     = "rkcy_zookeeper_pc"
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

%{ for i, ip in local.zookeeper_ips ~}
server.${i+1}=${ip}:2888:3888
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
    bastion_host        = "bastion0.${data.aws_route53_zone.rkcy_net.name}"
    bastion_private_key = file(var.ssh_key_path)

    user        = "ubuntu"
    host        = local.zookeeper_ips[count.index]
    private_key = file(var.ssh_key_path)
  }

  tags = {
    Name = "rkcy_inst_zookeeper_${count.index+1}"
  }
}

resource "aws_route53_record" "zookeeper_private" {
  count = var.zookeeper_count
  zone_id = data.aws_route53_zone.rkcy_net.zone_id
  name    = "zk${count.index+1}.local.${data.aws_route53_zone.rkcy_net.name}"
  type    = "A"
  ttl     = "300"
  records = [local.zookeeper_ips[count.index]]
}
#-------------------------------------------------------------------------------
# Zookeepers (END)
#-------------------------------------------------------------------------------


#-------------------------------------------------------------------------------
# Brokers
#-------------------------------------------------------------------------------
locals {
  kafka_ips = [for i in range(var.kafka_count) : "${cidrhost(local.sn_cidrs[i], 101)}"]
}

resource "aws_security_group" "rkcy_kafka" {
  name        = "rkcy_kafka"
  description = "Allow SSH and kafka inbound traffic"
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

resource "aws_network_interface" "kafka" {
  count = var.kafka_count
  subnet_id   = local.sn_ids[count.index]
  private_ips = [local.kafka_ips[count.index]]

  security_groups = [aws_security_group.rkcy_kafka.id]
}

resource "aws_placement_group" "kafka" {
  name     = "rkcy_kafka_pc"
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

  provisioner "remote-exec" {
    inline = [
      "sudo mkdir -p ${var.kafka_data_dir}"
      , "sudo chown -R kafka:kafka ${var.kafka_data_dir}"

      , <<EOF
sudo bash -c 'cat > /etc/systemd/system/kafka.service << EOF
[Unit]
Description=Apache Kafka server (broker)
Documentation=http://kafka.apache.org/documentation.html
Requires=network.target remote-fs.target
After=network.target remote-fs.target kafka.service

[Service]
Type=simple
User=kafka
Group=kafka
Environment=JAVA_HOME=/usr/lib/jvm/java-11-openjdk-amd64
ExecStart=/opt/kafka/bin/kafka-server-start.sh ${var.kafka_properties_file}
ExecStop=/opt/kafka/bin/kafka-server-stop.sh

[Install]
WantedBy=multi-user.target
EOF'
EOF
      , <<EOF
sudo bash -c 'cat > ${var.kafka_properties_file} << EOF
# see kafka.server.KafkaConfig for additional details and defaults

############################# Server Basics #############################

# The id of the broker. This must be set to a unique integer for each broker.
broker.id=${count.index}

############################# Socket Server Settings #############################

# The address the socket server listens on. It will get the value returned from
# java.net.InetAddress.getCanonicalHostName() if not configured.
#   FORMAT:
#     listeners = listener_name://host_name:port
#   EXAMPLE:
#     listeners = PLAINTEXT://your.host.name:9092
listeners=PLAINTEXT://:9092

# Hostname and port the broker will advertise to producers and consumers. If not set,
# it uses the value for "listeners" if configured.  Otherwise, it will use the value
# returned from java.net.InetAddress.getCanonicalHostName().
advertised.listeners=PLAINTEXT://kf${count.index}.local.${data.aws_route53_zone.rkcy_net.name}:9092

# Maps listener names to security protocols, the default is for them to be the same. See the config documentation for more details
#listener.security.protocol.map=PLAINTEXT:PLAINTEXT,SSL:SSL,SASL_PLAINTEXT:SASL_PLAINTEXT,SASL_SSL:SASL_SSL

# The number of threads that the server uses for receiving requests from the network and sending responses to the network
num.network.threads=3

# The number of threads that the server uses for processing requests, which may include disk I/O
num.io.threads=8

# The send buffer (SO_SNDBUF) used by the socket server
socket.send.buffer.bytes=102400

# The receive buffer (SO_RCVBUF) used by the socket server
socket.receive.buffer.bytes=102400

# The maximum size of a request that the socket server will accept (protection against OOM)
socket.request.max.bytes=104857600


############################# Log Basics #############################

# A comma separated list of directories under which to store log files
log.dirs=${var.kafka_data_dir}

# The default number of log partitions per topic. More partitions allow greater
# parallelism for consumption, but this will also result in more files across
# the brokers.
num.partitions=1

# The number of threads per data directory to be used for log recovery at startup and flushing at shutdown.
# This value is recommended to be increased for installations with data dirs located in RAID array.
num.recovery.threads.per.data.dir=1

############################# Internal Topic Settings  #############################
# The replication factor for the group metadata internal topics "__consumer_offsets" and "__transaction_state"
# For anything other than development testing, a value greater than 1 is recommended to ensure availability such as 3.
offsets.topic.replication.factor=1
transaction.state.log.replication.factor=1
transaction.state.log.min.isr=1

############################# Log Flush Policy #############################

# Messages are immediately written to the filesystem but by default we only fsync() to sync
# the OS cache lazily. The following configurations control the flush of data to disk.
# There are a few important trade-offs here:
#    1. Durability: Unflushed data may be lost if you are not using replication.
#    2. Latency: Very large flush intervals may lead to latency spikes when the flush does occur as there will be a lot of data to flush.
#    3. Throughput: The flush is generally the most expensive operation, and a small flush interval may lead to excessive seeks.
# The settings below allow one to configure the flush policy to flush data after a period of time or
# every N messages (or both). This can be done globally and overridden on a per-topic basis.

# The number of messages to accept before forcing a flush of data to disk
#log.flush.interval.messages=10000

# The maximum amount of time a message can sit in a log before we force a flush
#log.flush.interval.ms=1000

############################# Log Retention Policy #############################

# The following configurations control the disposal of log segments. The policy can
# be set to delete segments after a period of time, or after a given size has accumulated.
# A segment will be deleted whenever *either* of these criteria are met. Deletion always happens
# from the end of the log.

# The minimum age of a log file to be eligible for deletion due to age
log.retention.hours=168

# A size-based retention policy for logs. Segments are pruned from the log unless the remaining
# segments drop below log.retention.bytes. Functions independently of log.retention.hours.
#log.retention.bytes=1073741824

# The maximum size of a log segment file. When this size is reached a new log segment will be created.
log.segment.bytes=1073741824

# The interval at which log segments are checked to see if they can be deleted according
# to the retention policies
log.retention.check.interval.ms=300000

############################# Zookeeper #############################

# Zookeeper connection string (see zookeeper docs for details).
# This is a comma separated host:port pairs, each corresponding to a zk
# server. e.g. "127.0.0.1:3000,127.0.0.1:3001,127.0.0.1:3002".
# You can also append an optional chroot string to the urls to specify the
# root directory for all kafka znodes.
zookeeper.connect=${join(":2181,", local.zookeeper_ips)}:2181

# Timeout in ms for connecting to zookeeper
zookeeper.connection.timeout.ms=18000


############################# Group Coordinator Settings #############################

# The following configuration specifies the time, in milliseconds, that the GroupCoordinator will delay the initial consumer rebalance.
# The rebalance will be further delayed by the value of group.initial.rebalance.delay.ms as new members join the group, up to a maximum of max.poll.interval.ms.
# The default value for this is 3 seconds.
# We override this to 0 here as it makes for a better out-of-the-box experience for development and testing.
# However, in production environments the default value of 3 seconds is more suitable as this will help to avoid unnecessary, and potentially expensive, rebalances during application startup.
group.initial.rebalance.delay.ms=0
EOF'
EOF
      , "sudo chown kafka:kafka ${var.kafka_properties_file}"
      , "sudo systemctl daemon-reload"

      , <<EOF
%{for ip in local.zookeeper_ips}
RET=1
while [ \$RET -ne 0]; do
  echo Trying zookeeper ${ip}:2181
  nc -z ${ip} 2181
  RET=\$?
done
echo Connected zookeeper ${ip}:2181
%{endfor}
EOF

      , "sleep 10"

      , "sudo systemctl start kafka"
      , "sudo systemctl enable kafka"
    ]
  }

  connection {
    type     = "ssh"

    bastion_user        = "ubuntu"
    bastion_host        = "bastion0.${data.aws_route53_zone.rkcy_net.name}"
    bastion_private_key = file(var.ssh_key_path)

    user        = "ubuntu"
    host        = local.kafka_ips[count.index]
    private_key = file(var.ssh_key_path)
  }

  tags = {
    Name = "rkcy_inst_kafka_${count.index}"
  }
}

resource "aws_route53_record" "kafka_private" {
  count = var.kafka_count
  zone_id = data.aws_route53_zone.rkcy_net.zone_id
  name    = "kf${count.index}.local.${data.aws_route53_zone.rkcy_net.name}"
  type    = "A"
  ttl     = "300"
  records = [local.kafka_ips[count.index]]
}
#-------------------------------------------------------------------------------
# Brokers (END)
#-------------------------------------------------------------------------------
