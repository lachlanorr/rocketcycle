#!/bin/bash
cd "$(dirname "$0")"

createdb -h 127.0.0.1 -p 5432 -U postgres rpg
sqitch deploy
