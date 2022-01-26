variable "hostname" {
  type = string
}

variable "ssh_key_path" {
  type = string
}

variable "dev_public_ip" {
  type = string
}

variable "postgresql_hosts" {
  type = list
}

variable "kafka_cluster" {
  type = string
}

variable "kafka_hosts" {
  type = list
}

variable "otelcol_endpoint" {
  type = string
}
