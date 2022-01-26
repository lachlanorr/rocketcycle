resource "null_resource" "prometheus_provisioner" {
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
    content = templatefile("${path.module}/prometheus.service.tpl", {
      balancer_url = var.balancer_url
      prometheus_port = var.prometheus_port
    })
    destination = "/home/ubuntu/prometheus.service"
  }

  provisioner "file" {
    content = templatefile("${path.module}/prometheus.yml.tpl", {
      jobs = concat(var.jobs, [
        {
          name = "prometheus",
          targets = [for host in var.prometheus_hosts: "${host}:9100"]
          relabel = [
            {
              source_labels = ["__address__"]
              regex: "([^\\.]+).*"
              target_label = "instance"
              replacement = "$${1}"
            },
          ]
        },
        {
          name = "grafana",
          targets = [for host in var.grafana_hosts: "${host}:9100"]
          relabel = [
            {
              source_labels = ["__address__"]
              regex: "([^\\.]+).*"
              target_label = "instance"
              replacement = "$${1}"
            },
          ]
        },
      ])
    })
    destination = "/home/ubuntu/prometheus.yml"
  }

  provisioner "remote-exec" {
    inline = [
      <<EOF
sudo mv /home/ubuntu/prometheus.service /etc/systemd/system/prometheus.service
sudo mv /home/ubuntu/prometheus.yml /opt/prometheus/prometheus.yml
sudo systemctl daemon-reload
sudo systemctl start prometheus
sudo systemctl enable prometheus
EOF
    ]
  }

  connection {
    type     = "ssh"

    bastion_user        = "ubuntu"
    bastion_host        = var.bastion_ip
    bastion_private_key = file(var.ssh_key_path)

    user        = "ubuntu"
    host        = var.prometheus_ip
    private_key = file(var.ssh_key_path)
  }
}

