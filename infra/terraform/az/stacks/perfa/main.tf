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

variable "az_image_resource_group" {
  type = string
}

module "rkcy" {
  source = "../../modules/rkcy"

  cidr_block = "10.0.0.0/16"
  image_resource_group = var.az_image_resource_group
  stack = basename(abspath(path.module))
  dns_zone = "rkcy.us"
  public = true
}
