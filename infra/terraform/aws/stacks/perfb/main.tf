terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
      version = "~> 3.73"
    }
  }

  required_version = ">= 1.0.4"
}

provider "aws" {
  profile = "default"
  region = "us-east-2"
}

module "rkcy" {
  source = "../../modules/rkcy"

  cidr_block = "10.1.0.0/16"
  stack = basename(abspath(path.module))
  dns_zone = var.aws_domain
}
