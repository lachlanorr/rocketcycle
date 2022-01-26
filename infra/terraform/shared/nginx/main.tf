resource "null_resource" "nginx_provisioner" {
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
    bastion_host        = var.bastion_ip
    bastion_private_key = file(var.ssh_key_path)

    user        = "ubuntu"
    host        = var.nginx_ip
    private_key = file(var.ssh_key_path)
  }
}
