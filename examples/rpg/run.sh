#!/bin/bash
cd "$(dirname "$0")"

./init_db.sh

./rpg platform replace -e dev
./rpg config replace -e dev
./rpg run -d -e dev
