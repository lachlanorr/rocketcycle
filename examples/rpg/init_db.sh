#!/bin/bash
cd "$(dirname "$0")"

SCRIPT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
cd ${SCRIPT_DIR}/../../examples/rpg/db

createdb rpg -h 127.0.0.1 -p 5432 -U postgres
sqitch deploy --target db:pg://postgres@127.0.0.1:5432/rpg

createdb rpg -h 127.0.0.1 -p 5433 -U postgres
sqitch deploy --target db:pg://postgres@127.0.0.1:5433/rpg
