
receivers:
  otlp:
    protocols:
      grpc:
      http:

  # Collect own metrics
  prometheus:
    config:
      scrape_configs:
      - job_name: 'otel_collector'
        scrape_interval: 5s
        static_configs:
        - targets: ['127.0.0.1:8888']

processors:
  batch:

exporters:
  logging:
    logLevel: debug
  prometheus:
    endpoint: "0.0.0.0:9999"
  jaeger:
    endpoint: "${jaeger_collector}"
    insecure: true

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters:
        - jaeger
        #- logging

    metrics:
      receivers: [otlp, prometheus]
      processors: [batch]
      exporters:
        - prometheus
        #- logging
