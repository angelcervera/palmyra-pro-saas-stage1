#!/bin/bash
set -euo pipefail

HOST=${1:-postgres}
PORT=${2:-5432}
shift 2 || true

echo "[wait-for-postgres] waiting for ${HOST}:${PORT}"
until nc -z "${HOST}" "${PORT}" >/dev/null 2>&1; do
  echo "[wait-for-postgres] not ready, retrying..."
  sleep 2
done
echo "[wait-for-postgres] ready"

if [[ $# -gt 0 ]]; then
  exec "$@"
fi
