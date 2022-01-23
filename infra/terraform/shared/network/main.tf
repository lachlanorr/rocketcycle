resource "null_resource" "bastion_provisioner" {
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
    source = var.ssh_key_path
    destination = "~/.ssh/id_rsa"
  }

  provisioner "remote-exec" {
    inline = [
      "chmod 600 ~/.ssh/id_rsa"
    ]
  }

  connection {
    type     = "ssh"
    user     = "ubuntu"
    host     = var.bastion_ip
    private_key = file(var.ssh_key_path)
  }
}
