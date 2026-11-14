# 01 - PLC to cloud: store and forward

Work in progress.

```
Windows (plant)                 Edge                     Gateway                  Hub (Mac)
Factory I/O + CODESYS  --OPC UA-->  connector + NATS  --leaf-->  NATS  --leaf-->  NATS, InfluxDB, Grafana
soft PLC, OPC UA server             (RPi 1 later)                 (RPi 2 later)
```

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
