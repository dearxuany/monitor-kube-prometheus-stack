#! /bin/bash

#docker pull quay.io/prometheus/node-exporter:v0.18.1

docker run -d -p 9102:9102 \
  --net="host" \
  --pid="host" \
  -v "/:/host:ro,rslave" \
  quay.io/prometheus/node-exporter:v0.18.1 \
  --path.rootfs=/host \
  --web.listen-address=:9102

curl http://localhost:9102/metrics
