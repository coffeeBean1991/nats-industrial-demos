#!/usr/bin/env bash
# Stop the demo stacks: edge first, then gateway, then hub.
#
#   ./scripts/down.sh               # stop and remove containers, keep data
#   ./scripts/down.sh --wipe        # also delete all data (JetStream streams,
#                                   # InfluxDB, Grafana); asks for confirmation
#   ./scripts/down.sh --wipe -y     # same, without asking
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

wipe=false
yes=false
for arg in "$@"; do
  case "$arg" in
    --wipe) wipe=true ;;
    -y|--yes) yes=true ;;
    *) echo "usage: $0 [--wipe] [-y]" >&2; exit 1 ;;
  esac
done

flags="--remove-orphans"
if $wipe; then
  if ! $yes; then
    printf "This deletes the JetStream streams on all tiers, InfluxDB and Grafana data. Continue? [y/N] "
    read -r answer
    case "$answer" in y|Y|yes) ;; *) echo "aborted"; exit 1 ;; esac
  fi
  flags="$flags -v"
fi

for stack in edge gateway hub; do
  echo "==> stopping $stack"
  (cd "$ROOT/$stack" && docker compose down $flags)
done

if $wipe; then echo "All stacks stopped and their data deleted."; else echo "All stacks stopped. Data kept; use --wipe to delete it."; fi
