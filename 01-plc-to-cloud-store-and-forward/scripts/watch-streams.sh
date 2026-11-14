#!/usr/bin/env bash
# Live view of store and forward: message count of each tier's stream, how
# far each sourced stream lags behind the tier below, and how many messages
# the InfluxDB writer still has to process. A tier that can't be reached
# shows DOWN.
#
#   ./scripts/watch-streams.sh          # refresh every 2 s
#   ./scripts/watch-streams.sh 1        # refresh every 1 s
set -uo pipefail
interval="${1:-2}"

# value of the first "<field>": <number> in a JSON document on stdin
field() { grep -m1 "\"$1\"" | tr -dc '0-9'; }

stream_json() { docker exec "nats-box-$1" nats --timeout 1s stream info "$2" --json 2>/dev/null; }

row() { # tier stream show_lag
  local json msgs lag
  json="$(stream_json "$1" "$2")"
  if [ -z "$json" ]; then
    printf '%-15s %12s %12s\n' "$2" "DOWN" "-"
    return
  fi
  msgs="$(printf '%s' "$json" | field messages)"
  lag="-"
  if [ "$3" = "yes" ]; then lag="$(printf '%s' "$json" | field lag)"; fi
  printf '%-15s %12s %12s\n' "$2" "${msgs:-0}" "${lag:-?}"
}

while true; do
  screen="$(
    echo "$(date +%T)   store and forward watcher   (Ctrl-C to stop)"
    echo
    printf '%-15s %12s %12s\n' "STREAM" "MESSAGES" "SOURCE LAG"
    row edge    PLANT_EDGE    no
    row gateway PLANT_GATEWAY yes
    row hub     PLANT_HUB     yes
    echo
    pending="$(docker exec nats-box-hub nats --timeout 1s consumer info PLANT_HUB influxdb-writer --json 2>/dev/null | field num_pending)"
    printf '%-27s %12s\n' "InfluxDB writer pending" "${pending:-DOWN}"
  )"
  clear
  echo "$screen"
  sleep "$interval"
done
