# Nginx Prometheus Exporter
Prometheus 监控 nginx 有三种方法：
* nginx 扩展模块 ngx_http_stub_status_module 配合 nginx-prometheus-exporter
  
  ngx_http_stub_status_module 是比较常使用的 Nginx 状态监控模块，zabbix 对于 nginx 监控数据的收集也是基于此模块，主要收集基本的链接状态数据，能满足基本使用。
    ```
    # curl https://domainname.com/nginx_status
    Active connections: 17
    server accepts handled requests
    2537486 2537486 6960209
    Reading: 0 Writing: 3 Waiting: 14
    ``` 
  ngx_http_stub_status_module 输出格式并不是 prometheus 收集的数据格式，故需要配合 nginx-prometheus-exporter 来抓取数据并转换成 prometheus 可读取的格式。
  https://github.com/nginxinc/nginx-prometheus-exporter
  
* nginx 扩展模块 ngx_http_api_module  配合 nginx-prometheus-exporter

  使用 nginx-prometheus-exporter 的 NGINX Plus 收集模式，需要 nginx 配置有 ngx_http_api_module 开放 API 收集接口，数据收集内容很丰富。https://nginx.org/en/docs/http/ngx_http_api_module.html#api
  
* nginx 扩展模块 nginx_module_vts 配合 nginx-vts-exporter

  nginx-module-vts 为 nginx 扩展模块，监控内容相较于 ngx_http_stub_status_module 更加丰富，提供时延、IP等监控数据的统计。nginx-vts-exporter 主要用于收集Nginx的监控数据，并给Prometheus提供监控接口，默认端口号9913。

实际上，由于此前使用 zabbix 监控已安装过 ngx_http_stub_status_module，使用 ngx_http_api_module 或 nginx_module_vts 需要重新动态编译 nginx 模块，而且对于 nginx 的访问、时延统计已在日志服务对 nginx 日志进行分析实现，故此处使用第一种方式，只关注 nginx 的链接状态。

## nginx-prometheus-exporter 部署方案
此处将 nginx-prometheus-exporter 部署在 kubernetes 中，收集于 kubernetes 集群外部署在主机上的 nginx 数据。
### ngx_http_stub_status_module 
查看 nginx 已安装的模块
```
# nginx -V
nginx version: IWS/2.3.3
built by gcc 4.8.5 20150623 (Red Hat 4.8.5-39) (GCC)
built with OpenSSL 1.0.2k-fips  26 Jan 2017
TLS SNI support enabled
configure arguments: --prefix=/sdata/usr/local/nginx --user=nginx --group=nginx --with-http_stub_status_module --with-http_ssl_module --with-http_realip_module
```
确认 http_stub_status_module 已安装后，新增开放数据收集接口 /nginx_status 接口。

其中 hostname 可换成 IP 或主机名，因为是对应 nginx 节点的，故不应使用具体域名。

此处需要对 /nginx_status 做访问控制，由于 nginx-prometheus-exporter 在 k8s 内 pod IP 不是固定的，故允许 pod 分配网段访问。
```
# cat nginx_status_vpc.conf
server {
    listen 80;
    server_name hostname;

    access_log /sdata/var/log/nginx/nginx-status_access.log main;
    error_log /sdata/var/log/nginx/nginx-status_error.log error;

    location /nginx_status {
       stub_status on;
       allow 127.0.0.1;
       allow 172.11.0.0/16;
       deny all;
    }
}
```
重新加载 nginx 配置，访问 /nginx_status 接口
```
# curl -s http://$(hostname)/nginx_status
Active connections: 84
server accepts handled requests
46108827 46108827 85521487
Reading: 0 Writing: 2 Waiting: 82
```


### nginx-prometheus-exporter 
kubernetes 使用 deployment 部署 nginx-prometheus-exporter 实例。
```
kubectl create -f nginx-exporter-deployment.yaml
```
nginx-prometheus-exporter 在 k8s 内通过 nginx.scrape-uri 定义的地址抓取 k8s 集群外的 nginx 实例 /nginx_status 接口信息。
```
# cat nginx-exporter-deployment.yaml
... 
    spec:
      containers:
      - args:
        - -nginx.scrape-uri=http://hostname:80/nginx_status
```
由于一个 nginx-exporter-deployment 只能配置一个 nginx.scrape-uri 地址，故一个 nginx 实例就需要对应一个  nginx-exporter-deployment，无法像  nginx-ingress-controller 直接使用 dp 来拓展副本。

多实例 nginx 需要部署多个 nginx-exporter-deployment，但 nginx-exporter-service 只需一个，虽然 service 负载的多个 nginx-exporter-deployment 不是完全相同，但收集数据只需要将两个 dp 看作一样的进行轮询，从 metrics 返回的数据内容让 prometheus 识别区分不同的 nginx 实例即可。
```
kubectl create -f nginx-exporter-service.yaml
```
其中，不同 dp 的 name 必须独立，但 dp port name 除外，dp port name 用于关联 service port 故所有 dp 必须相同。
不同 db 用于关联 service 的 labels 也必须相同，以便让多个 dp 可以关联到一个 service 下。
```
apiVersion: v1
kind: Service
metadata:
  name: nginx-exporter
  labels:
    k8s-app: nginx-exporter
  namespace: monitoring
spec:
  ports:
  - name: nginx-exporter
    port: 9113
    targetPort: 9113
    protocol: TCP
  selector:
    k8s-app: nginx-exporter
  type: ClusterIP
  clusterIP: None
```
nginx-exporter 会把 /nginx_status 接口内容转换为 prometheus 可读取格式
```
curl http://nginx-exporter.monitoring.svc.cluster.local:9113/metrics

# HELP nginx_connections_accepted Accepted client connections
# TYPE nginx_connections_accepted counter
nginx_connections_accepted 2.72324e+06
# HELP nginx_connections_active Active client connections
# TYPE nginx_connections_active gauge
nginx_connections_active 23
# HELP nginx_connections_handled Handled client connections
# TYPE nginx_connections_handled counter
nginx_connections_handled 2.72324e+06
# HELP nginx_connections_reading Connections where NGINX is reading the request header
# TYPE nginx_connections_reading gauge
nginx_connections_reading 0
# HELP nginx_connections_waiting Idle client connections
# TYPE nginx_connections_waiting gauge
nginx_connections_waiting 6
# HELP nginx_connections_writing Connections where NGINX is writing the response back to the client
# TYPE nginx_connections_writing gauge
nginx_connections_writing 17
# HELP nginx_http_requests_total Total http requests
# TYPE nginx_http_requests_total counter
nginx_http_requests_total 2.403544e+06
# HELP nginx_up Status of the last metric scrape
# TYPE nginx_up gauge
nginx_up 1
# HELP nginxexporter_build_info Exporter build information
# TYPE nginxexporter_build_info gauge
nginxexporter_build_info{commit="5f88afbd906baae02edfbab4f5715e06d88538a0",date="2021-03-22T20:16:09Z",version="0.9.0"} 1
```
nginx-exporter-servicemonitor 也是只需要一个即可，通过 labels 直接关联 service。
```
kubectl create -f nginx-exporter-servicemonitor.yaml
```
由于 metrics 信息通过 nginx-exporter 进行中转，故 prometheus 拿到的 instance IP 并不是 nginx 实例的真实 IP 而是 nginx-exporter 的 podIP。
```
monitoring/nginx-exporter/0 (3/3 up) 

Endpoint	State	Labels	Last Scrape	Scrape Duration	Error
http://172.18.141.206:9113/metrics
UP	container="alihn1-prd-nginx-03-metrics" endpoint="nginx-exporter" instance="172.18.141.206:9113" job="nginx-exporter" namespace="monitoring" pod="alihn1-prd-nginx-03-metrics-f564f74d8-wp75z" service="nginx-exporter"	9.444s ago	9.998ms	

http://172.18.141.233:9113/metrics
UP	container="alihn1-prd-nginx-02-metrics" endpoint="nginx-exporter" instance="172.18.141.233:9113" job="nginx-exporter" namespace="monitoring" pod="alihn1-prd-nginx-02-metrics-6597bcd949-87wnm" service="nginx-exporter"	715ms ago	4.826ms	

http://172.18.141.234:9113/metrics
UP	container="alihn1-prd-nginx-01-metrics" endpoint="nginx-exporter" instance="172.18.141.234:9113" job="nginx-exporter" namespace="monitoring" pod="alihn1-prd-nginx-01-metrics-57fcf4f9d6-df7c6" service="nginx-exporter"	4.35s ago	2.489ms	
```
比如说，如果使用 container name 来做区分，nginx-exporter-deployment 使用 2 个副本的情况下，每个 pod 下 container name 依然是相同的，不影响数据的筛选，但会因为 instance 为 nginx-exporter 的 podIP 而被拆分为两条数据。
但这其实也可以看作数据抓取间隔不同的两近似的监控数据曲线，而且通常状况下 exporter 可以使用单实例，重启机制可满足需求。
```
monitoring/nginx-exporter/0 (4/4 up) 
Endpoint	State	Labels	Last Scrape	Scrape Duration	Error

http://172.18.141.206:9113/metrics
UP	container="alihn1-prd-nginx-03-metrics" endpoint="nginx-exporter" instance="172.18.141.206:9113" job="nginx-exporter" namespace="monitoring" pod="alihn1-prd-nginx-03-metrics-f564f74d8-wp75z" service="nginx-exporter"	14.904s ago	8.549ms	

http://172.18.141.233:9113/metrics
UP	container="alihn1-prd-nginx-02-metrics" endpoint="nginx-exporter" instance="172.18.141.233:9113" job="nginx-exporter" namespace="monitoring" pod="alihn1-prd-nginx-02-metrics-6597bcd949-87wnm" service="nginx-exporter"	6.516s ago	9.112ms	

http://172.18.141.234:9113/metrics
UP	container="alihn1-prd-nginx-01-metrics" endpoint="nginx-exporter" instance="172.18.141.234:9113" job="nginx-exporter" namespace="monitoring" pod="alihn1-prd-nginx-01-metrics-57fcf4f9d6-df7c6" service="nginx-exporter"	7.063s ago	2.135ms	

http://172.18.142.3:9113/metrics
UP	container="alihn1-prd-nginx-03-metrics" endpoint="nginx-exporter" instance="172.18.142.3:9113" job="nginx-exporter" namespace="monitoring" pod="alihn1-prd-nginx-03-metrics-f564f74d8-96hpp" service="nginx-exporter"	198ms ago	7.512ms	
```
可以考虑再 servicemonitor 中使用 relabelings 将 instance 替换成真实的 nginx 实例 IP。
https://cloud.tencent.com/document/product/1416/55995
```
# nginx-exporter 的默认 relabelings
- job_name: monitoring/nginx-exporter/0
  honor_labels: true
  honor_timestamps: true
  scrape_interval: 15s
  scrape_timeout: 10s
  metrics_path: /metrics
  scheme: http
  kubernetes_sd_configs:
  - role: endpoints
  relabel_configs:
  - source_labels: [__meta_kubernetes_service_label_k8s_app]
    separator: ;
    regex: nginx-exporter
    replacement: $1
    action: keep
  - source_labels: [__meta_kubernetes_endpoint_port_name]
    separator: ;
    regex: nginx-exporter
    replacement: $1
    action: keep
  - source_labels: [__meta_kubernetes_endpoint_address_target_kind, __meta_kubernetes_endpoint_address_target_name]
    separator: ;
    regex: Node;(.*)
    target_label: node
    replacement: ${1}
    action: replace
  - source_labels: [__meta_kubernetes_endpoint_address_target_kind, __meta_kubernetes_endpoint_address_target_name]
    separator: ;
    regex: Pod;(.*)
    target_label: pod
    replacement: ${1}
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    separator: ;
    regex: (.*)
    target_label: namespace
    replacement: $1
    action: replace
  - source_labels: [__meta_kubernetes_service_name]
    separator: ;
    regex: (.*)
    target_label: service
    replacement: $1
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    separator: ;
    regex: (.*)
    target_label: pod
    replacement: $1
    action: replace
  - source_labels: [__meta_kubernetes_pod_container_name]
    separator: ;
    regex: (.*)
    target_label: container
    replacement: $1
    action: replace
  - source_labels: [__meta_kubernetes_service_name]
    separator: ;
    regex: (.*)
    target_label: job
    replacement: ${1}
    action: replace
  - separator: ;
    regex: (.*)
    target_label: endpoint
    replacement: nginx-exporter
    action: replace
```
servicemoitor 启用 relabelings 后会将 prometheus label instance 值替换为 dp labels target-instance-id 的值。
实际上，也即是将 instance 默认显示的 nginx-exporter 实例地址替换为了目标监控 nginx 实例ID，更容易在数据上做聚合区分。
```
  - source_labels: [__meta_kubernetes_pod_label_target_instance_id]
    separator: ;
    regex: (.*)
    target_label: instance
    replacement: $1
    action: replace
```
修改后 prometheus target
```
monitoring/nginx-exporter/0 (3/3 up) 
Endpoint	State	Labels	Last Scrape	Scrape Duration	Error

http://172.18.141.233:9113/metrics
UP	container="nginx-exporter" endpoint="nginx-exporter" instance="alihn1-prd-nginx-01" job="nginx-exporter" namespace="monitoring" pod="alihn1-prd-nginx-01-metrics-7d7745464c-9kjqf" service="nginx-exporter"	10.428s ago	2.41ms	

http://172.18.142.11:9113/metrics
UP	container="nginx-exporter" endpoint="nginx-exporter" instance="alihn1-prd-nginx-02" job="nginx-exporter" namespace="monitoring" pod="alihn1-prd-nginx-02-metrics-fb6965f86-glblc" service="nginx-exporter"	5.705s ago	2.657ms	

http://172.18.141.234:9113/metrics
UP	container="nginx-exporter" endpoint="nginx-exporter" instance="alihn1-prd-nginx-03" job="nginx-exporter" namespace="monitoring" pod="alihn1-prd-nginx-03-metrics-5c8bcdb85c-9ns9k" service="nginx-exporter"
```
grafana 数据筛选修改为 intance 字段来对数据进行聚合和筛选区分 nginx 实例。
```
increase(nginx_http_requests_total{instance=~"$instance.*"}[1m])
``` 