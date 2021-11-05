#! /bin/bash
docker load -i grafana_7.1.0.tar
docker load -i kube-rbac-proxy_v0.4.1.tar
docker load -i kube-state-metrics_v1.9.5.tar
docker load -i node-exporter_v0.18.1.tar
docker load -i prometheus-config-reloader_v0.42.1.tar
docker load -i prometheus-operator_v0.42.1.tar
docker load -i prometheus_v2.20.0.tar