#!/bin/bash
cd "$(dirname "$0")"

sqitch revert -y
sqitch deploy
