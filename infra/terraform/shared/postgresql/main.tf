resource "null_resource" "postgresql_provisioner" {
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
    content = templatefile("${path.module}/postgresql.conf.tpl", {})
    destination = "/home/ubuntu/postgresql.conf"
  }

  provisioner "file" {
    content = templatefile("${path.module}/pg_hba.conf.tpl", {})
    destination = "/home/ubuntu/pg_hba.conf"
  }

  provisioner "file" {
    content = templatefile("${path.module}/pg_ident.conf.tpl", {})
    destination = "/home/ubuntu/pg_ident.conf"
  }

  provisioner "remote-exec" {
    inline = [
      <<EOF
# backup original config files
sudo mv /etc/postgresql/12/main/postgresql.conf /etc/postgresql/12/main/postgresql.conf.orig
sudo mv /etc/postgresql/12/main/pg_hba.conf /etc/postgresql/12/main/pg_hba.conf.orig
sudo mv /etc/postgresql/12/main/pg_ident.conf /etc/postgresql/12/main/pg_ident.conf.orig

sudo mv /home/ubuntu/postgresql.conf /etc/postgresql/12/main/
sudo mv /home/ubuntu/pg_hba.conf /etc/postgresql/12/main/
sudo mv /home/ubuntu/pg_ident.conf /etc/postgresql/12/main/

sudo chown postgres:postgres /etc/postgresql/12/main/postgresql.conf
sudo chmod 644 /etc/postgresql/12/main/postgresql.conf
sudo chown postgres:postgres /etc/postgresql/12/main/pg_hba.conf
sudo chmod 640 /etc/postgresql/12/main/pg_hba.conf
sudo chown postgres:postgres /etc/postgresql/12/main/pg_ident.conf
sudo chmod 640 /etc/postgresql/12/main/pg_ident.conf

sudo systemctl restart postgresql
EOF
    ]
  }

  connection {
    type     = "ssh"

    bastion_user        = "ubuntu"
    bastion_host        = var.bastion_ip
    bastion_private_key = file(var.ssh_key_path)

    user        = "ubuntu"
    host        = var.postgresql_ip
    private_key = file(var.ssh_key_path)
  }
}
