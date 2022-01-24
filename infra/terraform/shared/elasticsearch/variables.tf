variable "hostname" {
  type = string
}

variable "bastion_ip" {
  type = string
}

variable "ssh_key_path" {
  type = string
}

variable "stack" {
  type = string
}

variable "elasticsearch_index" {
  type = number
}

variable "elasticsearch_ips" {
  type = list
}

variable "elasticsearch_nodes" {
  type = list
}

variable "elasticsearch_rack" {
  type = string
}
