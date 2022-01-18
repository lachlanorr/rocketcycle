#!/bin/bash
cd /code/rocketcycle/examples/rpg/db

createdb -h ${postgresql_hosts[0]} -p 5432 -U postgres rpg
sqitch deploy --target db:pg://postgres@${postgresql_hosts[0]}:5432/rpg
