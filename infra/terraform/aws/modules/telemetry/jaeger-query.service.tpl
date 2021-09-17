[Unit]
Description=Jaeger Query
Documentation=https://www.jaegertracing.io/
Requires=network.target
After=network.target

[Service]
Type=simple
User=telem
Group=telem
Environment=SPAN_STORAGE_TYPE=elasticsearch
ExecStart=/usr/local/bin/jaeger-query --es.server-urls=${join(",", elasticsearch_urls)}
ExecStop=pgrep -f jaeger-query | xargs kill

[Install]
WantedBy=multi-user.target
