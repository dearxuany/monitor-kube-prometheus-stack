 # kube-prometheus exporter 服务自动发现
 prometheus 集群的部署方式直接影响监控数据收集 exporter 部署方式。
 
 现阶段，因为以监控 k8s 集群为前提，直接使用官方适配 k8s 集群监控分发的 kube-prometheus-operator 将 prometheus-stack 部署在 k8s 中，该 operator 相对于普通 prometheus-operator 多了一些为 k8s 预定制的监控规则与组件。
 
 按照 kube-prometheus-operator 官方维护人员的回答，kube-prometheus-operator 允许以添加 additionalScrapeConfigs 方式配置外部监控项，而更好的方法是直接使用 k8s 监控 CRD 服务自动发现。
 
 https://github.com/prometheus-operator/kube-prometheus/issues/379
 
> The additional scrape configs feature is just if you want to add raw Prometheus config, this is only meant as a last escape if no other functionality allows you to do what you are trying to do. You can feel free to manage additional ServiceMonitor objects as you prefer, you can do that via jsonnet or any other mechanism you like. For example we have ServiceMonitors deployed as part of each application, that way each application can own their monitoring configuration.

## Kubernetes CRD ServiceMonitor
ServiceMonitor 是 一个Kubernetes自定义资源 CRD，该资源描述了 Prometheus Server 的 Target 列表，Operator 会监听这个资源的变化来动态的更新 Prometheus Server 的 Scrape targets 并让 prometheus server 去 reload 配置。
该资源主要通过 Selector 来依据 Labels 选取对应的 Service 的 pod(endpoints)，并让 Prometheus Server 通过 Service 去 pull 监控目标的 metrics 信息，metrics信息要在 http 的 url 输出符合 metrics 格式的信息，ServiceMonitor 也可以定义目标的 metrics 的 url。

ServiceMonitors 通过 k8s 内部的 service 配合 labels 来进行服务发现和通信，但是在 k8s 外部的 exporter 不存在 k8s service，故需要提供一个 service 和 Endpoints 给外部 exporter 才能让 ServiceMonitor 可与其通信，进而被 prometheus 发现且定期向目标 metrics 接口发送请求、收集监控数据。


### metrics 抓取对象
- k8s cluster 内服务
  - 不需要 exporter
    - 特点：
      - 服务部署在 k8s 内
      - 自带 metrics 接口，可直接被 prometheus 抓取识别
      - instance ip 为目标实例真实 ip
    - 样例：nginx-ingress-controller、springboot-actuator
    - 数据流：目标实例 pod -> service -> servicemonitor -> prometheus -> grafana
- k8s cluster 外服务
  - 不需要 exporter
    - 特点：
      - 服务部署在 k8s 外
      - 自带 metrics 接口，可直接被 prometheus 抓取识别
    - 样例：canal
    - 数据流：目标实例 -> 自定义 endpoint -> service -> servicemonitor -> prometheus -> grafana
  - 需要 exporter
    - 总体特点：
      - 服务部署在 k8s 外
      - 需要 exporter 收集目标实例监控数据，且数据需要 exporter 转换后才能被 prometheus 抓取识别
      - instance ip 为非目标实例真实 ip，需要 k8s labels 配合 relabelings 转换，以便区分数据来源
    - exporter 在 k8s 内
      - 特点：目标实例有自定义的一套监控数据提供模板、接口、格式
      - 样例：kafka-exporter、nginx-exporter、mysql-exporter
      - 数据流：目标实例 -> exporter pod -> exporter service -> servicemonitor -> prometheus -> grafana
    - exporter 在 k8s 外
      - 特点：exporter 必须和目标实例部署在同一主机，直接通过 exporter 收集监控数据
      - 样例：node-exporter
      - 数据流：目标实例 exporter -> endpoint -> servicemonitor -> prometheus -> grafana
      
具体的部署方式可参照 exporter-deploy 中每个 exporter 中的 README.md 文档。


### prometheus-server 服务自动发现规则
查看 kube-prometheus-operator 部署的 prometheus 实例 manifest
```
# kubectl get prometheus k8s -o yaml -n monitoring
apiVersion: monitoring.coreos.com/v1
kind: Prometheus
metadata:
  annotations:
...
spec:
  securityContext:
    fsGroup: 2000
    runAsNonRoot: true
    runAsUser: 1000
  serviceAccountName: prometheus-k8s
  serviceMonitorNamespaceSelector: {}
  serviceMonitorSelector: {}
  storage:
    volumeClaimTemplate:
      metadata: {}
      spec:
        accessModes:
        - ReadWriteOnce
        resources:
          requests:
            storage: 1T
        storageClassName: alicloud-nas-prometheus
      status: {}
  version: v2.20.0
```
其中，prometheus 对 serviceMonitor 资源自动发现配置为 {} 即空，此时 prometheus 将自动发现 k8s cluster 中所有 namespace 中的所有 serviceMonitor 资源，直接使用即可。
```
  serviceMonitorNamespaceSelector: {}
  serviceMonitorSelector: {}
```
以 nginx-exporter 的 servicemonitor 为例
```
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: nginx-exporter
  namespace: monitoring
  labels:
    k8s-app: nginx-exporter
spec:
  endpoints:
  - interval: 15s
    port: nginx-exporter
    #path: /metrics
    honorLabels: true
    relabelings:
      - action: replace
        separator: ;
        sourceLabels: [ __meta_kubernetes_pod_label_target_instance_id ]
        regex: (.*)
        targetLabel: instance
  namespaceSelector:
    any: true
  selector:
    matchLabels:
      k8s-app: nginx-exporter
```
此处设置了 namespaceSelector 为 any: true，即允许所有 namespaces 中的 prometheus 都可以发现该 nginx-exporter。 
如果有需要最好还是在 ServiceMonitor 中设置一下发现限制，比如自建的 prometheus 在 monitoring 命名空间中，而阿里云自带的在 arms-prom 中，为了不让阿里云的 prometheus 抓到自定义监控数据而产生大量费用，建议限制 ServiceMonitor 的 namespaceSelector 仅为 monitoring。
```
  namespaceSelector:
    any: true
```
