resource "null_resource" "grafana_provisioner" {
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
    content = templatefile("${path.module}/grafana.ini.tpl", {
      balancer_url = var.balancer_external_url
      grafana_port = var.grafana_port
    })
    destination = "/home/ubuntu/grafana.ini"
  }

  provisioner "file" {
    content = templatefile("${path.module}/datasources.yaml.tpl", {
      balancer_url = var.balancer_internal_url
    })
    destination = "/home/ubuntu/datasources.yaml"
  }

  provisioner "file" {
    content = templatefile("${path.module}/dashboards.yaml.tpl", {})
    destination = "/home/ubuntu/dashboards.yaml"
  }

  provisioner "file" {
    source = "${path.module}/dashboards"
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
    bastion_host        = var.bastion_ip
    bastion_private_key = file(var.ssh_key_path)

    user        = "ubuntu"
    host        = var.grafana_ip
    private_key = file(var.ssh_key_path)
  }
}
