#!/bin/sh
set -eu

if [ -f /vault/secrets/backend-env ]; then
  set -a
  . /vault/secrets/backend-env
  set +a
fi

exec /app/search-api
