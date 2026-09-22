#!/bin/sh
# Loads .env (copy .env.example -> .env and fill in your values) and runs
# the app. See README.md for what each variable does.
set -eu

if [ -f .env ]; then
  set -a
  . ./.env
  set +a
fi

exec go run ./cmd/printcost
