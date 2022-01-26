resource "null_resource" "dev_provisioner" {
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
      "${path.module}/init_db.sh.tpl",
      {
        postgresql_hosts = var.postgresql_hosts
      })
    destination = "/code/rocketcycle/build/bin/init_db_aws.sh"
  }

  provisioner "file" {
    content = templatefile(
      "${path.module}/platform.json.tpl",
      {
        kafka_cluster = var.kafka_cluster
        kafka_hosts = var.kafka_hosts
        postgresql_hosts = var.postgresql_hosts
      })
    destination = "/code/rocketcycle/build/bin/platform_aws.json"
  }

  provisioner "file" {
    content = templatefile(
      "${path.module}/run_aws.sh.tpl",
      {
        kafka_hosts = var.kafka_hosts
        otelcol_endpoint = var.otelcol_endpoint
      })
    destination = "/code/rocketcycle/build/bin/run_aws.sh"
  }

  provisioner "remote-exec" {
    inline = [
      <<EOF
chmod +x /code/rocketcycle/build/bin/init_db_aws.sh
chmod +x /code/rocketcycle/build/bin/run_aws.sh

ulimit -n 10240
EOF
    ]
  }

  connection {
    type     = "ssh"
    user     = "ubuntu"
    host     = var.dev_public_ip
    private_key = file(var.ssh_key_path)
  }
}
