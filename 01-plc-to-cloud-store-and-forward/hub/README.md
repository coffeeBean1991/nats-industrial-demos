# Hub stack

"The cloud". The gateway connects here as a leaf node, and the `PLANT_HUB`
stream sources `PLANT_GATEWAY` through the gateway's JetStream domain
(`$JS.gateway.API`). A durable consumer writes every reading into InfluxDB,
acking only after InfluxDB accepts it, and Grafana shows the plant and the
health of NATS on all three tiers.

| Service | Role |
|---------|------|
| `nats-hub` | NATS server, JetStream domain `hub`; leaf port for the gateway |
| `nats-box-hub` | Creates `PLANT_HUB` at startup, then available for `nats` CLI use |
| `nats-connector` | Durable pull consumer `influxdb-writer` on `PLANT_HUB` -> InfluxDB (measurement `levelcontrol`) |
| `influxdb` | Time series storage, bucket `plant` |
| `grafana` | Dashboards: Level Control, NATS Servers, NATS JetStream |
| `prometheus` | Receives the NATS metrics of all tiers by remote write |
| `nats-exporter-hub` | NATS monitoring as Prometheus metrics |
| `fluentbit-hub` | Receives edge + gateway metrics, adds its own, writes to Prometheus |

Readings are written with the PLC's own timestamp (OPC UA source
timestamp), so data that arrives late after an outage lands at the right
time in Grafana and the gap fills in.

## Run

```bash
cp .env.example .env
docker compose up -d --build
```

| What | Where |
|------|-------|
| Grafana | http://localhost:3000 (admin / admin; the Level Control dashboard is the home page) |
| InfluxDB | http://localhost:8086 (admin / admin12345) |
| Prometheus | http://localhost:9090 |

## Check

```bash
docker exec nats-box-hub nats stream info PLANT_HUB                     # Source Lag 0, count matches the other tiers
docker exec nats-box-hub nats consumer info PLANT_HUB influxdb-writer   # Unprocessed messages: 0
docker compose logs -f nats-connector                                   # "wrote level_meter=8.27 to influxdb"
```
