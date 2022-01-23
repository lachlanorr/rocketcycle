resource "null_resource" "zookeeper_provisioner" {
  #---------------------------------------------------------
  # node_exporter
  #---------------------------------------------------------
  provisioner "remote-exec" {
    inline = ["sudo hostnamectl set-hostname ${var.hostname}"]
  }
  provisioner "file" {
    content = templatefile("${path.module}/../../node_exporter_install.sh", {})
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
    content = templatefile("${path.module}/zookeeper.service.tpl", {})
    destination = "/home/ubuntu/zookeeper.service"
  }

  provisioner "file" {
    content = templatefile(
      "${path.module}/zookeeper.properties.tpl",
      {
        zookeeper_ips = var.zookeeper_ips
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
sudo bash -c 'echo ${var.zookeeper_index+1} > /data/zookeeper/myid'
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
    bastion_host        = var.bastion_ip
    bastion_private_key = file(var.ssh_key_path)

    user        = "ubuntu"
    host        = var.zookeeper_ips[var.zookeeper_index]
    private_key = file(var.ssh_key_path)
  }
}
