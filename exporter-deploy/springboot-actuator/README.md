# kube-prometheus-operator springboot actuator
此处 manifest 只做 demo 示范，实际使用 jenkins CD 与应用发布一起进行部署管理。

### jenkins project-parameter
依赖 jenkins project-parameter 两个参数，可独立对各个项目进行配置。

以下为两参数的默认值，默认开启 springboot-actuator 监控，应用 metrics 路径默认为 /actuator/prometheus。
```
Common:
ServiceMonitor_Status=yes
ServiceMonitor_Path=/actuator/prometheus
```
ServiceMonitor_Path 需根据项目实际使用跳转路径配置，大部分为 /actuator/prometheus，有少部分有特殊跳转的确认具体链路。

### jenkins CD-job
jenkins 部署 CD 添加 project-parameter 两个对应参数并设置默认值，各位环境 CD-job 均需添加。
当 project-parameter 未配置对应参数时，jenkins 自动使用 CD 面板配置的参数值。
```
# 字符参数

名称 ServiceMonitor_Status
默认值 yes

名称 ServiceMonitor_Path
默认值 /actuator/prometheus
```
jenkins CD-job 构建将参数值传给 helm
```
if [[ "${ServiceMonitor_Status}" == 'yes' ]];then
    helm_args="${helm_args} --set serviceMonitor.enable=true --set serviceMonitor.path=${ServiceMonitor_Path}"
fi
```

### kubernetes helm chart
helm 根据项目参数渲染 k8s chart 部署 serviceMonitor
```
# cat values.yaml
serviceMonitor:
  enable: false
  path: /actuator/prometheus


# cat templates/servicemonitor.yaml
{{- if .Values.serviceMonitor.enable }}
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  #  填写一个唯一名称
  name: {{ include "name" . }}
  #  填写目标命名空间
  namespace: {{ .Values.dp.namespace }}
spec:
  endpoints:
  - interval: 10s
    #  填写service.yaml中Prometheus Exporter对应的Port的Name字段的值
    port: http
    #  填写Prometheus Exporter对应的Path的值
    path: {{ .Values.serviceMonitor.path }}
  namespaceSelector:
    any: true
    #  Nginx Demo的命名空间
  selector:
    matchLabels:
      #  填写service.yaml的Label字段的值以定位目标service.yaml
      app: {{ include "name" . }}
{{ end }}
```
helm 根据模板渲染好的 manifest 伴随应用 app 同时发布，除特殊情况外无需手动重复配置。