variable "hostname" {
  type = string
}

variable "bastion_ip" {
  type = string
}

variable "ssh_key_path" {
  type = string
}

variable "nginx_ip" {
  type = string
}

variable "routes" {
  type = list
}
