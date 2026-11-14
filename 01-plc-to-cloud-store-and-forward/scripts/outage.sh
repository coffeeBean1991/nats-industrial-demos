#!/usr/bin/env bash
# Simulate a network outage between two tiers, then restore it.
#
#   ./scripts/outage.sh edge-gateway cut       # the site gateway goes down
#   ./scripts/outage.sh edge-gateway restore
#   ./scripts/outage.sh gateway-hub cut        # the cloud (hub) goes down
#   ./scripts/outage.sh gateway-hub restore
#
# With every stack on one machine, a link is cut by stopping the NATS server
# on the upstream side of it. Stopping nats-gateway also cuts gateway-hub;
# on separate devices, pull the cable instead.
set -euo pipefail

usage() { echo "usage: $0 <edge-gateway|gateway-hub> <cut|restore>" >&2; exit 1; }
[ $# -eq 2 ] || usage

case "$1" in
  edge-gateway) server=nats-gateway; buffer="PLANT_EDGE (edge)" ;;
  gateway-hub)  server=nats-hub;     buffer="PLANT_GATEWAY (gateway)" ;;
  *) usage ;;
esac

case "$2" in
  cut)
    docker stop "$server" >/dev/null
    echo "$(date +%T)  link $1 CUT ($server stopped)"
    echo "  The plant keeps running; readings are buffered in $buffer."
    echo "  Watch: ./scripts/watch-streams.sh   Restore: $0 $1 restore"
    ;;
  restore)
    docker start "$server" >/dev/null
    echo "$(date +%T)  link $1 RESTORED ($server started)"
    echo "  Sources resume from their last sequence; the gap in Grafana backfills."
    ;;
  *) usage ;;
esac
