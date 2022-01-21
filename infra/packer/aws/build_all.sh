#!/bin/bash
cd "$(dirname "$0")"

packer build bastion.pkr.hcl
packer build dev.pkr.hcl
packer build docker.pkr.hcl
packer build elasticsearch.pkr.hcl
packer build hashi.pkr.hcl
packer build kafka.pkr.hcl
packer build metrics.pkr.hcl
packer build nginx.pkr.hcl
packer build postgresql.pkr.hcl
packer build telemetry.pkr.hcl
