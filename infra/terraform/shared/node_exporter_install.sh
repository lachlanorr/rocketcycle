cat << EOF >> ~/node_exporter.service
[Unit]
Description=Node Exporter
Documentation=https://prometheus.io/
Requires=network.target
After=network.target

[Service]
Type=simple
User=metrics
Group=metrics
ExecStart=/usr/local/bin/node_exporter
ExecStop=pgrep -f node_exporter | xargs kill

[Install]
WantedBy=multi-user.target
EOF

sudo mv ~/node_exporter.service /etc/systemd/system/node_exporter.service
sudo chown root:root /etc/systemd/system/node_exporter.service
sudo systemctl daemon-reload
sudo systemctl enable node_exporter
sudo systemctl start node_exporter
