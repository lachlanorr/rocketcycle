variable "hostname" {
  type = string
}

variable "bastion_ip" {
  type = string
}

variable "ssh_key_path" {
  type = string
}

variable "public" {
  type = bool
}

variable "kafka_index" {
  type = number
}

variable "kafka_rack" {
  type = string
}

variable "kafka_internal_ips" {
  type = list
}

variable "kafka_internal_host" {
  type = string
}

variable "kafka_external_host" {
  type = string
}

variable "zookeeper_ips" {
  type = list
}
