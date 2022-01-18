#!/bin/bash
cd "$(dirname "$0")"

./init_db_aws.sh

./rpg platform replace -e dev --platform_file_path=platform_aws.json --otelcol_endpoint=${otelcol_endpoint} --admin_brokers=${kafka_hosts[0]}
./rpg config replace -e dev --config_file_path=config.json --otelcol_endpoint=${otelcol_endpoint} --admin_brokers=${kafka_hosts[0]}
./rpg run -e dev --otelcol_endpoint=${otelcol_endpoint} --admin_brokers=${kafka_hosts[0]}
