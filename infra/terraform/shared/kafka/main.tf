resource "null_resource" "kafka_provisioner" {
  #---------------------------------------------------------
  # node_exporter
  #---------------------------------------------------------
  provisioner "remote-exec" {
    inline = ["sudo hostnamectl set-hostname ${var.hostname}"]
  }
  provisioner "file" {
    content = templatefile("${path.module}/../node_exporter_install.sh", {})
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
    content = templatefile("${path.module}/kafka.service.tpl", {})
    destination = "/home/ubuntu/kafka.service"
  }

  provisioner "file" {
    content = templatefile(
      "${path.module}/kafka.properties.tpl",
      {
        public = var.public,
        idx = var.kafka_index,
        kafka_rack = var.kafka_rack,
        kafka_internal_ips = var.kafka_internal_ips,
        kafka_internal_host = var.kafka_internal_host,
        kafka_external_host = var.kafka_external_host,
        zookeeper_ips = var.zookeeper_ips,
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

%{for ip in var.zookeeper_ips}
RET=1
while [ $RET -ne 0 ]; do
  echo Trying zookeeper ${ip}:2181
  nc -z ${ip} 2181
  RET=$?
  sleep 2
done
echo Connected zookeeper ${ip}:2181
%{endfor}

sudo systemctl start kafka
sudo systemctl enable kafka
EOF
    ]
  }

  connection {
    type     = "ssh"

    bastion_user        = "ubuntu"
    bastion_host        = var.bastion_ip
    bastion_private_key = file(var.ssh_key_path)

    user        = "ubuntu"
    host        = var.kafka_internal_ips[var.kafka_index]
    private_key = file(var.ssh_key_path)
  }
}
