# Kafka Prometheus Exporter
Prometheus 监控 kafka 主要通过 kafka-exporter 访问 kafka 原生开放 api 来获取监控数据并转换成 prometheus 可识别的数据结构。

可通过 kafka-exporter 来监控 kafka 节点状态、不同 topic 及 GID 间的消费关系、分区状况、消息堆积状态。

https://github.com/danielqsj/kafka_exporter

## kafka-exporter 部署方案
将 kafka-exporter 以 deployment 形式部署在 k8s 内，须在 deployment 来以 args 形式将 kafka 地址传给 kafka-exporter。
由于访问 kafka 集群一个节点即可调用到 kafka 监控 API，故一个 kafka 集群对应一个 kafka-exporter-deployment 即可收集到该 kafka 集群的 metrics 信息，也可一个 kafka 节点对应部署一个 dp。

```
# cat kafka-exporter-deployment.yaml

    spec:
      containers:
      - args:
        - --kafka.server=172.18.30.110:9092
        - --zookeeper.server=172.18.30.110:2181

```
另外 kafka-exporter 的 docker images 需固定版本，否则会出现传入参数和版本不一致报错无法启动，现阶段使用 tag 为 latest，实际为 v1.4.3。
https://hub.docker.com/r/danielqsj/kafka-exporter/tags
```
      image: danielqsj/kafka-exporter:latest
        imagePullPolicy: IfNotPresent
```

和 nginx-exporter 类似，dp port name 用于关联 service port 故所有 dp、service 必须相同，db 用于关联 service 的 labels 也必须相同，以便让多个 dp 可以关联到一个 service 下。
kafka-exporter-servicemonitor 也是只需要一个即可，通过 labels 直接关联 service。
```
kubectl create -f kafka-exporter-deployment.yaml
kubectl create -f kafka-exporter-service.yaml
kubectl create -f kafka-exporter-servicemonitor.yaml
```
由于经过 kafka-exporter 转换，prometheus 默认收到的 instance 值并不是 kafka 节点的真实 IP 值，不利于在 grafana 中进行节点的筛选查询。

此处将使用在 kafka-exporter-deployment 中定义的 labels target-instance-id 来替换 prometheus labels instance。

https://cloud.tencent.com/document/product/1416/55995#service-monitor

```
# kafka-exporter-deployment.yaml 中添加

  labels:
    k8s-app: kafka-exporter
    target-instance-id: alihn1-prd-kafka-01


# kafka-exporter-servicemonitor.yaml 中添加

    relabelings:
    - action: replace
      separator: ;
      sourceLabels: [__meta_kubernetes_pod_label_target_instance_id]
      regex: (.*)
      targetLabel: instance
```
其中 targetLabel 不一定是需要在 prometheus label 中已有的 labels，也可以是自定义的一个新的 label 字段。

此处为了和无需 exporter 的应用统一，直接将 instance 值替换为 __meta_kubernetes_pod_label_target_instance_id。

需注意 deployment labels 中的 target-instance-id 不能带有特殊字符且释义清晰，能用于数据聚合中直接区分监控目标实例。
```
monitoring/kafka-exporter/0 (1/1 up) 
Endpoint	State	Labels	Last Scrape	Scrape Duration	Error
http://172.18.142.11:9308/metrics
UP	container="alihn1-prd-kafka-01-metrics" endpoint="kafka-exporter" instance="alihn1-prd-kafka-01" job="kafka-exporter" namespace="monitoring" pod="alihn1-prd-kafka-01-metrics-5f9b98777c-fkcv9" service="kafka-exporter"	1
```