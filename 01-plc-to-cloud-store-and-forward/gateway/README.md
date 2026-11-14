# Gateway stack

The aggregation tier at the IT/OT boundary. The edge connects here as a
leaf node, and the `PLANT_GATEWAY` stream sources `PLANT_EDGE` through the
edge's JetStream domain (`$JS.edge.API`). The gateway in turn connects to
the hub as a leaf node. While the uplink to the hub is down, the gateway
keeps storing; while the link from the edge is down, the edge does.

| Service | Role |
|---------|------|
| `nats-gateway` | NATS server, JetStream domain `gateway`; leaf port for the edge, leaf node of the hub |
| `nats-box-gateway` | Creates `PLANT_GATEWAY` at startup, then available for `nats` CLI use |
| `nats-exporter-gateway` | NATS monitoring as Prometheus metrics |
| `fluentbit-gateway` | Receives the edge metrics, adds its own, forwards both to the hub |

`PLANT_GATEWAY` has no subjects of its own on purpose. Plain NATS messages
also cross leaf node connections, so if it listened on `plant.>` as well, it
would store every reading twice: once from the subject and once from the
source.

## Run

```bash
cp .env.example .env
docker compose up -d
```

## Check

```bash
docker exec nats-box-gateway nats stream info PLANT_GATEWAY   # "Source Information": Lag 0, count matches PLANT_EDGE
curl -s localhost:8223/leafz | grep -E '"name"|"ip"'          # leaf nodes connected (edge, and the hub once it runs)
```

Until the hub stack is running, `nats-gateway` and `fluentbit-gateway` log
connection retries to the hub. That's expected.
