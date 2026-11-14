#!/usr/bin/env bash
# Start the demo stacks: hub first, then gateway, then edge, so each tier
# finds the one above it already listening.
#
#   ./scripts/up.sh                 # all three stacks
#   ./scripts/up.sh edge            # only the stacks you name
#
# A stack without a .env gets one copied from its .env.example.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

if [ $# -eq 0 ]; then set -- hub gateway edge; fi

for stack in "$@"; do
  dir="$ROOT/$stack"
  if [ ! -f "$dir/docker-compose.yml" ]; then
    echo "unknown stack: $stack (expected hub, gateway or edge)" >&2
    exit 1
  fi
  if [ ! -f "$dir/.env" ]; then
    cp "$dir/.env.example" "$dir/.env"
    echo "created $stack/.env from .env.example - check its values"
  fi
  echo "==> starting $stack"
  (cd "$dir" && docker compose up -d --build)
done

cat <<'MSG'

Stacks are up. Give the edge ~10 s to connect to the PLC, then:
  Grafana     http://localhost:3000   (Level Control dashboard)
  Watcher     ./scripts/watch-streams.sh
MSG
