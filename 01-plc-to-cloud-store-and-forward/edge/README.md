# Edge stack

Runs next to the PLC: reads the CODESYS tags over OPC UA and stores every
value change in the `PLANT_EDGE` JetStream stream, which the gateway sources
over a leaf node connection. While the uplink is down, the edge keeps
storing; nothing is lost.

| Service | Role |
|---------|------|
| `opcua-connector` | OPC UA subscription to the `FIO` tags, JetStream publish to `plant.levelcontrol.<tag>` |
| `nats-edge` | NATS server, JetStream domain `edge`, leaf node of the gateway |
| `nats-box-edge` | Creates `PLANT_EDGE` at startup, then available for `nats` CLI use |
| `nats-exporter-edge` | NATS monitoring as Prometheus metrics |
| `fluentbit-edge` | Forwards those metrics to the gateway's Fluent Bit |

## Run

```bash
cp .env.example .env      # then set OPCUA_ENDPOINT etc.
docker compose up -d --build
```

The tag list and subjects are in `opcua-connector/config.yaml`.

## Check

```bash
docker compose logs -f opcua-connector                       # "-> plant.levelcontrol.level_meter [PLANT_EDGE #123] 8.27"
docker exec nats-box-edge nats stream info PLANT_EDGE        # message count growing
docker exec nats-box-edge nats sub 'plant.>'                 # live values
```

Until the gateway stack is running, `nats-edge` and `fluentbit-edge` log
connection retries to the gateway. That's expected: the edge buffers
everything locally until the gateway is there.
