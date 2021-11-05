# monitor-prometheus-stack
monitor-prometheus-stack 仓库用户存储多环境 kubernetes prometheus stack 使用的 manifests yaml。

其中，prometheus 主体使用官方给 k8s 版本定制的 prometheus-operator 。相对于普通版本的 prometheus-operator，k8s 定制版已经包含了 k8s 集群监控需要的组件和监控规则，无需再额外部署。
kube-prometheus-release-0.6 项目 github 原地址 https://github.com/prometheus-operator/kube-prometheus/tree/release-0.6

此外，为对现有环境做适配，对官方的 manifests yaml 文件做了一些修改，主要包括但不限于：
* 变更 prometheus 数据轮替时间为 30 天
* k8s 增加阿里云 Nas CSI 插件，使用阿里云的 nas 做持久化存储，prd 使用性能型 nas，其余使用容量型
* 新增 k8s 内指定实例的部署节点，dev 和 prd 有指定部署节点，但 qas 和 uat 没有
* 修改一些 k8s api 访问权限配置，主要是 kube-state-metrics 对 k8s 集群的访问权限
* 修改 node-exporter 端口为 9102，解决和阿里云 node-exporter 端口冲突，全量接入后把阿里自带实例下线
* 新增 grafana、prometheus、alertmanager 访问页面 k8s nginx-ingress 证书、路由配置
* nginx-ingress 新增 http2 配置参数适配 grafana 对 prometheus 查询请求要求 
* grafana 默认数据源名称修改，方便 prd grafana 做多环境数据源聚合
* k8s 集群内外 exporter 服务自动发现，如网关入口、DB、MQ 等 metrics 数据收集
* 监控目标实例 target-instance-id 传输，grafana dashboard model 多环境、多实例聚合修改
* percona-monitoring-management server 部署，mysql、mongodb、postgres 接入 Query Analytics
* 调整 alertmanager 默认告警规则
```
# tree -L 2
.
├── db-percona-monitoring-management           # 多种 DB 性能、Query Analytics 监控平台部署
│    ├── pmm-server-service.yaml
│    ├── pmm-server-statefulset.yaml
│    └── README.md
├── docker-images                              # docker 镜像打包 tar 
├── exporter-deploy                            # k8s 集群外 exporter 服务自动发现配置
│    ├── canal-metrics
│    ├── kafka-exporter
│    ├── mysql-exporter
│    ├── nginx-exporter
│    ├── nginx-ingress-controller-metrics
│    ├── node-exporter
│    └── springboot-actuator                
├── kube-aliyun-nas-mount                      # 阿里云 nas storageClass、PVC 新建 
│    ├── alibaba-cloud-csi-driver-master       # k8s 支持阿里云 nas 的 CSI 插件
│    ├── dev
│    ├── prd
│    ├── qas
│    └── uat
├── kube-grafana-config                        # grafana 数据源修改配置
│    ├── dev
│    └── prd
├── kube-prometheus-release-0.6                # kube prometheus-operator 项目代码
│    ├── build.sh
│    ├── code-of-conduct.md
│    ├── DCO
│    ├── docs
│    ├── example.jsonnet
│    ├── examples
│    ├── experimental
│    ├── go.mod
│    ├── go.sum
│    ├── hack
│    ├── jsonnet
│    ├── jsonnetfile.json
│    ├── jsonnetfile.lock.json
│    ├── kustomization.yaml
│    ├── LICENSE
│    ├── Makefile
│    ├── manifests                             # kube prometheus-operator 部署 yaml 目录
│    ├── NOTICE
│    ├── OWNERS
│    ├── README.md
│    ├── scripts
│    ├── sync-to-internal-registry.jsonnet
│    ├── tests
│    └── test.sh
├── kube-prom-ingress                          # 访问页面 nginx-ingress 配置 
│    ├── dev
│    ├── prd
│    ├── qas
│    ├── secret
│    └── uat
├── README.md
└── src                                        # 插件源码包
    ├── alibaba-cloud-csi-driver-master.zip
    └── kube-prometheus-release-0.6.zip

27 directories, 19 files

```
## kube-prometheus-operator 部署
此处以 prd 阿里云 k8s 环境为例，dev 为自建 k8s 会有部分调整。
### 持久化存储
k8s 集群新建 namespace
```
# kubectl create ns monitoring    
```
持久化存储部署目录
```
cd kube-aliyun-nas-mount
```
k8s ns 中新建持久化存储资源，grafana 使用的是 NAS storageClass + dp PVC，prom 使用的是 NAS storageClass + StatefulSet PVC。

注意：若果是非阿里云托管的 k8s 自建集群，需要使用阿里云的 nas 做持久化存储，需先在自建集群内安装 CSI 插件。
```
# kubectl create -f grafana-storageClass.yaml -n monitoring    
storageclass.storage.k8s.io/alicloud-nas-grafana created

# kubectl create -f grafana-PVC.yaml -n monitoring    
persistentvolumeclaim/alicloud-nas-grafana-csi-pvc created

# kubectl create -f prometheus-promStorageClass.yaml -n monitoring    
storageclass.storage.k8s.io/alicloud-nas-prometheus created
```
查询 storageClass 和 pvc，其中 prom 是使用 statefulset 部署，故是在部署 prom 同时建立 pvc。
```
# kubectl get storageclass -n monitoring    
NAME                       PROVISIONER                       AGE
alicloud-nas-grafana       nasplugin.csi.alibabacloud.com    3m51s
alicloud-nas-prometheus    nasplugin.csi.alibabacloud.com    2m49s

# kubectl get pvc -n monitoring    
NAME                           STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS           AGE
alicloud-nas-grafana-csi-pvc   Bound    nas-7723d705-032b-4a6f-9e4e-59ce90b98dfc   200Gi      RWX            alicloud-nas-grafana   95s
```

### 指定实例部署 k8s 节点
添加节点标记，使用 pod 部署到指定节点，注意 dev 和 prd 有指定部署节点，但 qas 和 uat 没有。
```
# kubectl label node alihn1-prd-k8s-w30129.xuany node.tag/usage=ops    
node/alihn1-prd-k8s-w30129.xuany labeled
```

### prometheus-operator 部署
进入 prometheus-operator 部署目录
```
cd kube-prometheus-release-0.6  
```
#### 新建 K8S 监控 CRD
部署 k8s monitor 监控 CRD
```
# kubectl create -f manifests/setup    
```
因为此前已装过某些组件可能已经存在某些 CRD，直接使用即可，一共 7 个监控 CRD。
```
# kubectl get crd -n monitoring     |grep coreos
alertmanagers.monitoring.coreos.com                         2020-05-26T05:38:26Z
podmonitors.monitoring.coreos.com                           2021-05-08T09:13:33Z
probes.monitoring.coreos.com                                2020-12-11T21:28:14Z
prometheuses.monitoring.coreos.com                          2020-05-26T05:38:26Z
prometheusrules.monitoring.coreos.com                       2020-05-26T05:38:26Z
servicemonitors.monitoring.coreos.com                       2020-05-26T05:38:26Z
thanosrulers.monitoring.coreos.com                          2021-05-08T09:13:34Z
```
#### 部署 prometheus 及其组件
此处不要指定 namespace，有些组件需要 k8s 管理的读取权限，需要装在指定 ns 中，否则会安装失败。
```
# kubectl create -f manifests    
```
日常维护 prometheus 组件直接使用 kubectl apply 命令
```
# kubectl apply -f manifests/grafana-deployment.yaml    
```

#### 修复 kube-state-metrics 对 k8s api 读取权限问题
kube-state-metrics 报错日志
```
E0907 06:48:57.758886 1 reflector.go:156] pkg/mod/k8s.io/client-go@v0.0.0-20191109102209-3c0d1af94be5/tools/cache/reflector.go:108: Failed to list *v1.Job: jobs.batch is forbidden: User "system:serviceaccount:monitoring:kube-state-metrics" cannot list resource "jobs" in API group "batch" at the cluster scope
2021/9/7 下午2:48:57 E0907 06:48:57.759818 1 reflector.go:156] pkg/mod/k8s.io/client-go@v0.0.0-20191109102209-3c0d1af94be5/tools/cache/reflector.go:108: Failed to list *v1.DaemonSet: daemonsets.apps is forbidden: User "system:serviceaccount:monitoring:kube-state-metrics" cannot list resource "daemonsets" in API group "apps" at the cluster scope
```
kube-state-metrics 现在绑定的 clusterRole 不对，现在为阿里云使用的 rules 需修改为 prometheus-operator 定义的 clusterRole，需删除旧 clusterRole 重新新建。
```
# kubectl delete ClusterRoleBinding kube-state-metrics -n monitoring     
warning: deleting cluster-scoped resources, not scoped to the provided namespace
clusterrolebinding.rbac.authorization.k8s.io "kube-state-metrics" deleted

# kubectl create -f kube-state-metrics-clusterRoleBinding.yaml -n monitoring        
clusterrolebinding.rbac.authorization.k8s.io/kube-state-metrics created
```

#### 添加 k8s ingress 允许外部访问
k8s nginx-ingress 部署目录
```
cd kube-prom-ingress 
```
https secret 证书新建
```
略
```
nginx-ingress 路由及 http 强制跳转 https
```
略
```
nginx-configuration grafana http2 请求配置适配（启用 https 才需要）
```
http2-max-concurrent-streams: 128
http2-max-field-size: 32k
http2-max-header-size: 64k
large-client-header-buffers: 8 64k
```

#### grafana 默认数据源名称修改
进入 grafana 数据源修改目录
```
cd kube-grafana-config 
```
修改 grafana 默认数据源名称
```
# cat datasources.yaml
{
    "apiVersion": 1,
    "datasources": [
        {
            "access": "proxy",
            "editable": false,
            "name": "prd-prometheus",
            "orgId": 1,
            "type": "prometheus",
            "url": "http://prometheus-k8s.monitoring.svc:9090",
            "version": 1
        }
    ]
}

# kubectl create secret generic grafana-datasources-v2  --from-file=./datasources.yaml -n monitoring    
secret/grafana-datasources-v2 created

# cat datasources-v3.yaml
{
    "apiVersion": 1,
    "datasources": [
        {
            "access": "proxy",
            "editable": true,
            "name": "prometheus",
            "orgId": 1,
            "type": "prometheus",
            "url": "http://prometheus-k8s.monitoring.svc:9090",
            "version": 1
        }
    ]
}


# kubectl create secret generic grafana-datasources-v3  --from-file=./datasources-v3.yaml -n monitoring    
secret/grafana-datasources-v3 created
```
启用 grafana-datasources-v2 添加不可修改数据源 prd-prometheus
```
# vim grafana-deployment.yaml
...
      volumes:
      - persistentVolumeClaim:
          claimName: alicloud-nas-grafana-csi-pvc
        name: grafana-storage
      - name: grafana-datasources
        secret:
          secretName: grafana-datasources-v2


# kubectl apply -f grafana-deployment.yaml    
```
启用 grafana-datasources-v3 将默认 grafana 数据源 prometheus 改为可修改后，grafana 页面删除 prometheus 数据源
```
# vim grafana-deployment.yaml
...
      volumes:
      - persistentVolumeClaim:
          claimName: alicloud-nas-grafana-csi-pvc
        name: grafana-storage
      - name: grafana-datasources
        secret:
          secretName: grafana-datasources-v3


# kubectl apply -f grafana-deployment.yaml    
```
重新启用 grafana-datasources-v2 ，防止 grafana 重启读到旧的 secret 把默认 prometheus 数据源加回去。
```
# vim grafana-deployment.yaml
...
      volumes:
      - persistentVolumeClaim:
          claimName: alicloud-nas-grafana-csi-pvc
        name: grafana-storage
      - name: grafana-datasources
        secret:
          secretName: grafana-datasources-v2




# kubectl apply -f grafana-deployment.yaml    
```
执行最新操作后 grafana 容器里的数据源配置
```
bash-5.0$ cat /etc/grafana/provisioning/datasources/datasources.yaml
{
    "apiVersion": 1,
    "datasources": [
        {
            "access": "proxy",
            "editable": false,
            "name": "prd-prometheus",
            "orgId": 1,
            "type": "prometheus",
            "url": "http://prometheus-k8s.monitoring.svc:9090",
            "version": 1
        }
    ]
}
```

### prometheus/grafana 用户权限管理
暂时不使用多租户模式，将所有用户分配到默认 Main Org. 组织。
非管理用户使用 Editor Role 允许查询 metrics 并新建面板，并在已有统一汇聚面板中设置已由公用面板 Editor Role 权限为 View，只允许查询不允许修改。