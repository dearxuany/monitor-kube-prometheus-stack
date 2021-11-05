# NGINX Ingress Controller Metrics
默认情况下 nginx-ingress 的监控指标端口为 10254，监控路径为其下的 /metrics。
调整配置 nginx-ingress-controller 的配置文件，打开 service 及 pod 的 10254 端口。
https://kubernetes.github.io/ingress-nginx/user-guide/monitoring/

## rancher deploy
对于 rancher 来说，使用的是 DaemonSet 来部署 nginx-ingress 且以  node port 去暴露端口，没有 service，nginx-ingress-controller 的 pod IP 就是 k8s 节点的 nodeIP。

https://docs.rancher.cn/docs/rke/config-options/add-ons/ingress-controllers/_index/

修改 nginx-ingress-controller DaemonSet manifests yaml，主要是开放 metrics 以及开放节点端口 10254。
```
apiVersion: v1
kind: DaemonSet
spec:
  template:
    metadata:
      annotations:
   prometheus.io/scrape: "true"
   prometheus.io/port: "10254"
..
ports:
  - containerPort: 10254
    hostPort: 10254
    name: prometheus
    protocol: TCP
  ..
```
访问节点 10254 端口的 /metrics 接口
```
# curl http://10.0.0.95:10254/metrics
# HELP go_gc_duration_seconds A summary of the pause duration of garbage collection cycles.
# TYPE go_gc_duration_seconds summary
go_gc_duration_seconds{quantile="0"} 2.3237e-05
go_gc_duration_seconds{quantile="0.25"} 4.7288e-05
go_gc_duration_seconds{quantile="0.5"} 7.9446e-05

...

# TYPE nginx_ingress_controller_success counter
nginx_ingress_controller_success{controller_class="nginx",controller_namespace="ingress-nginx",controller_pod="nginx-ingress-controller-drc2w"} 1
# HELP process_cpu_seconds_total Total user and system CPU time spent in seconds.
# TYPE process_cpu_seconds_total counter
process_cpu_seconds_total 3.26
# HELP process_max_fds Maximum number of open file descriptors.
# TYPE process_max_fds gauge
process_max_fds 1.048576e+06
# HELP process_open_fds Number of open file descriptors.
# TYPE process_open_fds gauge
process_open_fds 29
# HELP process_resident_memory_bytes Resident memory size in bytes.
# TYPE process_resident_memory_bytes gauge
process_resident_memory_bytes 6.3041536e+07
# HELP process_start_time_seconds Start time of the process since unix epoch in seconds.
# TYPE process_start_time_seconds gauge
process_start_time_seconds 1.63366065564e+09
# HELP process_virtual_memory_bytes Virtual memory size in bytes.
# TYPE process_virtual_memory_bytes gauge
process_virtual_memory_bytes 7.5941888e+08
# HELP process_virtual_memory_max_bytes Maximum amount of virtual memory available in bytes.
# TYPE process_virtual_memory_max_bytes gauge
process_virtual_memory_max_bytes -1
# HELP promhttp_metric_handler_requests_in_flight Current number of scrapes being served.
# TYPE promhttp_metric_handler_requests_in_flight gauge
promhttp_metric_handler_requests_in_flight 1
# HELP promhttp_metric_handler_requests_total Total number of scrapes by HTTP status code.
# TYPE promhttp_metric_handler_requests_total counter
promhttp_metric_handler_requests_total{code="200"} 0
promhttp_metric_handler_requests_total{code="500"} 0
promhttp_metric_handler_requests_total{code="503"} 0
```
获取 nginx-ingress-controller 所在 k8s 节点 IP
```
# kubectl get pod -n ingress-nginx  --kubeconfig ~/.kubeconfig/dev/dev-admin.kubeconfig -o wide|awk '{print $6}'|sort -r
```
根据 IP 列表新建 endpoints 和 service，并配置 prometheus servicemonitor 服务自动发现
```
kubectl create -f rancher/nginx-ingress-controller-metrics-endpoints.yaml -n monitoring
kubectl create -f rancher/nginx-ingress-controller-metrics-service.yaml -n monitoring
kubectl create -f rancher/nginx-ingress-controller-metrics-servicemonitoring.yaml -n monitoring
```

## aliyun deploy
阿里云部署的 nginx-ingress-controller 使用的是 Deployment + LoadBalancer Service 方案部署，直接使用阿里云的 SLB 进行对 nginx-ingress pod 的负载均衡。

其中，nginx-ingress-controller service 允许区分绑定公网和私网的 LB 以区分来自内外网的流量，实现公私网分流及访问控制。

https://help.aliyun.com/document_detail/151524.html

### 启用 metrics
nginx-ingress-controller Deployment 添加 prometheus 注释
```
spec:
  template:
    metadata:
      annotations:
        prometheus.io/port: "10254"
        prometheus.io/scrape: "true"
...
ports:
  - containerPort: 10254
    name: prometheus
    protocol: TCP
```
由于 metrics 只需要在 k8s 集群内被 prometheus 访问到，不需要开放到 k8s 集群外，故对于 ingress-nginx 的 metrics service 并不需要绑定 SLB，只需直接使用 type 为 clusterIP 的 service 在 k8s 内部做负载均衡即可。

独立新建 nginx-ingress-controller Service 关联 dp 10254 端口开放 metrics 访问：
```
kubectl create -f aliyun/nginx-ingress-controller-metrics-service.yaml -n kube-system
```
跨 ns 访问 metrics 接口
```
# curl http://nginx-ingress-controller-metrics.kube-system.svc.cluster.local:10254/metrics
```
但由于 nginx-ingress dp 和 prometheus 不在同一 namespace，为实现跨 ns 的 service 访问，需要在 prometheus 所在的 ns 添加一个 type 为 externalName 的 service 来支持 servicemonitor 服务发现，当然也可以直接把 servicemonitor 部署在 nginx-ingress 所在的 ns 中。 
其中，需注意 ports 的名称，需要和 nginx-ingress dp 中的 ports name 对应。
```
kubectl create -f aliyun/nginx-ingress-controller-metrics-servicemonitoring.yaml -n kube-system
```
公网的 nginx-ingress metrics 配置和私网的相同，只需修改 k8s 资源名称和 labels 为 external-nginx-ingress 的即可。