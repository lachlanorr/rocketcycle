#!/bin/bash

SCRIPT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
cd ${SCRIPT_DIR}/../../examples/rpg/db

createdb -h postgresql-0.perfa.rkcy.net -p 5432 -U postgres rpg
sqitch deploy --target db:pg://postgres@postgresql-0.perfa.rkcy.net:5432/rpg
