apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: ${balancer_url}/prometheus
    isDefault: true
    version: 1
    editable: true
