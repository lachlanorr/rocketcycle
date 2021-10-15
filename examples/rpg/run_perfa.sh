#!/bin/bash
cd "$(dirname "$0")"

./init_db_perfa.sh

. ./perfa_env.sh

./rpg platform replace --platform_file_path=platform_perfa.json --otelcol_endpoint=edge.perfa.rkcy.net:4317 --admin_brokers=kafka-0.clusa.perfa.rkcy.net:9093
./rpg run --otelcol_endpoint=edge.perfa.rkcy.net:4317 --admin_brokers=kafka-0.clusa.perfa.rkcy.net:9093
