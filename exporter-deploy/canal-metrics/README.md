# kube-prometheus-operator ali-canal monitor
官方参考文档

https://github.com/alibaba/canal/issues/765

https://github.com/alibaba/canal/wiki/Prometheus-QuickStart

### canal-metrics
canal 启用 metrics 收集端口
```
# tcp bind ip
canal.ip =
# register ip to zookeeper
canal.register.ip =
canal.port = 11111
canal.metrics.pull.port = 11112
```
访问 metrics 端口
```
# curl localhost:11112

# HELP jvm_classes_loaded The number of classes that are currently loaded in the JVM
# TYPE jvm_classes_loaded gauge
jvm_classes_loaded 5559.0
# HELP jvm_classes_loaded_total The total number of classes that have been loaded since the JVM has started execution
# TYPE jvm_classes_loaded_total counter
jvm_classes_loaded_total 5559.0
# HELP jvm_classes_unloaded_total The total number of classes that have been unloaded since the JVM has started execution
# TYPE jvm_classes_unloaded_total counter
jvm_classes_unloaded_total 0.0
# HELP jvm_info JVM version info
# TYPE jvm_info gauge
jvm_info{version="1.8.0_171-b11",vendor="Oracle Corporation",runtime="Java(TM) SE Runtime Environment",} 1.0
# HELP canal_instance Canal instance
...
```
### kube-prometheus-operator 服务发现
canal 由于为数据库同步中间件，部署在 k8s 集群外，故需配置 endpoints 和 service 让 prometheus 可在 k8s 集群内通过 serviceMonitor 发现 canal 实例并采集数据。
```
kubectl apply -f canal-metrics-endpoints.yaml -n monitoring
kubectl apply -f canal-metrics-service.yaml -n monitoring
kubectl apply -f canal-metrics-servicemonitor.yaml -n monitoring
```
k8s 集群内跨 ns 访问路径
```
curl http://canal-metrics.monitoring.svc.cluster.local:11112
```

