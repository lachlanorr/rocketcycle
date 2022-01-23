output "balancer_internal_urls" {
  value = {
    edge = module.nginx_edge.balancer_internal_url
    app = module.nginx_app.balancer_internal_url
  }
}

output "balancer_external_urls" {
  value = {
    edge = module.nginx_edge.balancer_external_url
    app = module.nginx_app.balancer_external_url
  }
}

output "nginx_hosts" {
  value = {
    edge = sort(module.nginx_edge.nginx_hosts)
    app = sort(module.nginx_app.nginx_hosts)
  }
}
