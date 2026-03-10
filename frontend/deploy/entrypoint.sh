#!/bin/sh
set -eu

if [ -f /vault/secrets/frontend-env ]; then
  set -a
  . /vault/secrets/frontend-env
  set +a
fi

exec npm start
