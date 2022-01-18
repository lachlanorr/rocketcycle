[Unit]
Description=Prometheus
Documentation=https://prometheus.io/
Requires=network.target
After=network.target

[Service]
Type=simple
User=prometheus
Group=prometheus
WorkingDirectory=/opt/prometheus
ExecStart=/opt/prometheus/prometheus --config.file=/opt/prometheus/prometheus.yml --web.external-url=${ balancer_url }/prometheus --web.listen-address=:${ prometheus_port}
ExecStop=pgrep -f prometheus | xargs kill

[Install]
WantedBy=multi-user.target
