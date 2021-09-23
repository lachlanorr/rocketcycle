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
