#!/bin/bash
cd "$(dirname "$0")"

./init_db.sh

./rpg platform replace
./rpg config replace
./rpg run -d
