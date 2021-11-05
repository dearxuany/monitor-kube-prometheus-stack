# kube-prometheus-operator docker images
kube-prometheus-operator 使用到的 docker 镜像 tar 打包，直接在部署前导入 k8s node 可节省镜像拉取时间。
* 导出
```
#! /bin/bash
docker save -o prometheus-config-reloader_v0.42.1.tar  quay.io/prometheus-operator/prometheus-config-reloader           
docker save -o prometheus-operator_v0.42.1.tar quay.io/prometheus-operator/prometheus-operator     
docker save -o prometheus_v2.20.0.tar quay.io/prometheus/prometheus      
docker save -o kube-state-metrics_v1.9.5.tar quay.io/coreos/kube-state-metrics    
docker save -o node-exporter_v0.18.1.tar quay.io/prometheus/node-exporter         
docker save -o kube-rbac-proxy_v0.4.1.tar quay.io/coreos/kube-rbac-proxy    
docker save -o grafana_7.1.0.tar grafana/grafana
```
* 导入
```
#! /bin/bash
docker load -i grafana_7.1.0.tar
docker load -i kube-rbac-proxy_v0.4.1.tar
docker load -i kube-state-metrics_v1.9.5.tar
docker load -i node-exporter_v0.18.1.tar
docker load -i prometheus-config-reloader_v0.42.1.tar
docker load -i prometheus-operator_v0.42.1.tar
docker load -i prometheus_v2.20.0.tar
```