terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
      version = "~> 3.27"
    }
  }

  required_version = ">= 0.14.9"
}

provider "aws" {
  profile = "default"
  region = "us-east-2"
}

variable "aws_dns_zone" {}

module "rkcy" {
  source = "../../modules/rkcy"

  cidr_block = "10.0.0.0/16"
  stack = basename(abspath(path.module))
  dns_zone = var.aws_dns_zone
  public = true
}
