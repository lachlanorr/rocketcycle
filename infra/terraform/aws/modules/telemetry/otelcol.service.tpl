[Unit]
Description=Open Telemetry Collector
Documentation=https://opentelemetry.io/docs/
Requires=network.target
After=network.target

[Service]
Type=simple
User=telem
Groupr=telem
Environment=SPAN_STORAGE_TYPE=elasticsearch
ExecStart=/usr/local/bin/otelcol --config /etc/otelcol.yaml
ExecStop=pgrep -f otelcol | xargs kill

[Install]
WantedBy=multi-user.target
