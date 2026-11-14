# 01 - PLC to cloud: store and forward

Work in progress.

```
Windows (plant)                 Edge                     Gateway                  Hub (Mac)
Factory I/O + CODESYS  --OPC UA-->  connector + NATS  --leaf-->  NATS  --leaf-->  NATS, InfluxDB, Grafana
soft PLC, OPC UA server             (RPi 1 later)                 (RPi 2 later)
```

## Quick start

```bash
./scripts/up.sh                       # hub, gateway, edge (creates missing .env files from .env.example)
./scripts/watch-streams.sh            # live message counts, source lag, writer backlog
./scripts/outage.sh edge-gateway cut  # simulate an outage; "restore" to bring it back
./scripts/down.sh                     # stop everything (--wipe also deletes the data)
```

Set `OPCUA_ENDPOINT` in `edge/.env` to your CODESYS machine first.

| Folder | Contents |
|--------|----------|
| `plant/codesys/` | CODESYS program and the `FIO` global variable list |
| `plant/factoryio/` | Factory I/O scene notes |
| `edge/` | Docker Compose stack for the edge device |
| `gateway/` | Docker Compose stack for the gateway |
| `hub/` | Docker Compose stack for the hub |
| `scripts/` | Stream watcher and outage helpers |
| `docs/` | Diagrams and video notes |

Each stack is configured through its own `.env` file, so the same stack runs
on one machine for development or on separate devices without code changes.
