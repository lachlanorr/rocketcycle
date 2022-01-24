resource "null_resource" "elasticsearch_provisioner" {
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
    content = templatefile(
      "${path.module}/elasticsearch.yml.tpl",
      {
        stack = var.stack
        idx = var.elasticsearch_index
        elasticsearch_ips = var.elasticsearch_ips
        elasticsearch_nodes = var.elasticsearch_nodes
        elasticsearch_rack = var.elasticsearch_rack
      })
    destination = "/home/ubuntu/elasticsearch.yml"
  }
  provisioner "remote-exec" {
    inline = [
      <<EOF
# backup original config file
sudo mv /etc/elasticsearch/elasticsearch.yml /etc/elasticsearch/elasticsearch.yml.orig
sudo mv /home/ubuntu/elasticsearch.yml /etc/elasticsearch/elasticsearch.yml

sudo chown root:elasticsearch /etc/elasticsearch/elasticsearch.yml
sudo chmod 660 /etc/elasticsearch/elasticsearch.yml

sudo systemctl start elasticsearch
sudo systemctl enable elasticsearch

EOF
    ]
  }

  connection {
    type     = "ssh"

    bastion_user        = "ubuntu"
    bastion_host        = var.bastion_ip
    bastion_private_key = file(var.ssh_key_path)

    user        = "ubuntu"
    host        = var.elasticsearch_ips[var.elasticsearch_index]
    private_key = file(var.ssh_key_path)
  }
}
