#!/bin/bash
cd "$(dirname "$0")"

createdb -h postgresql-0.perfa.local.rkcy.net -p 5432 -U postgres rpg
sqitch deploy --target rpg_perfa
