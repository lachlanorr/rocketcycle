#!/bin/bash
cd "$(dirname "$0")"

sqitch revert -y --target db:pg://postgres@127.0.0.1:5432/rpg
sqitch deploy --target db:pg://postgres@127.0.0.1:5432/rpg

sqitch revert -y --target db:pg://postgres@127.0.0.1:5433/rpg
sqitch deploy --target db:pg://postgres@127.0.0.1:5433/rpg
