[Unit]
Description=Jaeger Collector
Documentation=https://www.jaegertracing.io/
Requires=network.target
After=network.target

[Service]
Type=simple
User=telem
Group=telem
Environment=SPAN_STORAGE_TYPE=elasticsearch
ExecStart=/usr/local/bin/jaeger-collector --es.server-urls=${join(",", elasticsearch_urls)}
ExecStop=pgrep -f jaeger-collector | xargs kill

[Install]
WantedBy=multi-user.target
