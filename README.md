# NATS industrial demos

Hands-on demos of NATS and JetStream in industrial settings: from the plant
floor (OPC UA, PLCs) through edge and gateway tiers to the cloud. Each folder
is one video and runs on its own: go into the folder and follow its README.

| # | Demo | What it shows |
|---|------|---------------|
| 00 | [Intro: OPC UA to NATS edge-to-cloud](00-intro-opcua-nats/) | Simulated OPC UA sensors, an edge -> gateway -> hub NATS leafnode chain with JetStream sourcing, InfluxDB and Grafana, store and forward during outages |
| 01 | [PLC to cloud: store and forward](01-plc-to-cloud-store-and-forward/) | Factory I/O + CODESYS soft PLC as the plant, separate edge, gateway and hub stacks (Raspberry Pi ready). Work in progress |
