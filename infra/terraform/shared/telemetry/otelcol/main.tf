resource "null_resource" "otelcol_provisioner" {
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
    content = templatefile("${path.module}/otelcol.service.tpl", {
      elasticsearch_urls = var.elasticsearch_urls
    })
    destination = "/home/ubuntu/otelcol.service"
  }

  provisioner "file" {
    content = templatefile("${path.module}/otelcol.yaml.tpl", {
      jaeger_collector = "${ var.nginx_telem_host }:${ var.jaeger_collector_port }"
    })
    destination = "/home/ubuntu/otelcol.yaml"
  }

  provisioner "remote-exec" {
    inline = [
      <<EOF
sudo mv /home/ubuntu/otelcol.service /etc/systemd/system/otelcol.service
sudo mv /home/ubuntu/otelcol.yaml /etc/otelcol.yaml
sudo chown telem:telem /etc/otelcol.yaml
sudo chmod 660 /etc/otelcol.yaml

sudo systemctl daemon-reload
sudo systemctl start otelcol
sudo systemctl enable otelcol
EOF
    ]
  }

  connection {
    type     = "ssh"

    bastion_user        = "ubuntu"
    bastion_host        = var.bastion_ip
    bastion_private_key = file(var.ssh_key_path)

    user        = "ubuntu"
    host        = var.otelcol_ip
    private_key = file(var.ssh_key_path)
  }
}
