terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "=2.91.0"
    }
  }
}

provider "azurerm" {
  features {}
}

variable "image_resource_group" {
  type = string
}

variable "stack" {
  type = string
}

variable "dns_zone" {
  type = string
}

variable "cidr_block" {
  type = string
}

variable "public" {
  type = bool
}

module "network" {
  source = "../../modules/network"

  cidr_block = var.cidr_block
  image_resource_group = var.image_resource_group
  stack = var.stack
  dns_zone = var.dns_zone
}
