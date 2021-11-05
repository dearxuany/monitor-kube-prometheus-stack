# Percona Monitoring and Management 
Percona Monitoring and Management 简称 PMM，是一款支持对多种大型数据库集群进行全面监控的老牌监控工具。

对比 mysql-exporter，PMM 支持收集并分析更加细致的 db 性能指标及 SQL 执行指标。

比较关键的是， PMM 支持 Query Analytics 能收集分析每条 SQL 执行内容及性能状态，并支持在 grafana 上进行可视化查询分析，非常有助于帮助应用程序的 debug 与优化，其功能相当于阿里云的收费 SQL 洞察功能。

https://www.percona.com/software/database-tools/percona-monitoring-and-management

## PMM 架构
PMM 分为 server 与 client 结构，client 部署在数据库节点主机上，向 server 注册。

PMM Client：
* pmm-admin：提供命令行交互界面管理pmm client，包括新增、删除数据库实例等
  
* pmm-mysql-query-0：管理 mysql QAN代理的服务，从数据库实例搜集查询性能数据并发送到pmm server上的QAN API

* pmm-mongdb-query-0：管理mongdb QAN代理的服务

* node_exporter：搜集系统性能数据，基于prometheus exporter

* mysqld_exporter：mysql性能搜集
  
* mongodb_exporter: mongodb性能搜集

* proxysql_exporter：ProxySQL性能数据搜集

PMM Server:

* QAN（Query Analytics）：分析mysql数据库的查询性能
  + qan api：后端存储和获取由agent采集的查询性能数据
  + qan web：提供数据展示
* metrics monitor：提供mysql和mongodb的历史性能数据查询
  + prometheus：第三方的时序数据库，连接到pmm client的exporter并汇集数据，consul给pmm client提供api用于远程list，添加删除prometheus上的配置主机，并存储监控的元数据
  + grafana：第三方的图形展示界面
* Orchestrator：提供mysql复制的拓扑工具和图形界面


## PMM 部署方案
因为部署在 k8s 中，可以考虑使用 helm 部署 PMM Server、使用 yum 部署 PMM Client。
https://www.percona.com/blog/2020/07/23/using-percona-kubernetes-operators-with-percona-monitoring-and-management/

由于官方提供 PMM Server helm chart 已自带 prometheus 和 grafana 部署，但由于已有这些资源，再独立维护 PMM 的 promethues 和 grafana 比较麻烦。

此处使用 percona 官方提供的 pmm-server docker images 来部署 PMM Server，该镜像内已包含 PMM Server 所有组件，直接使用 manifest 拉起该容器并开放访问端口即可。
此方案无需再独立去维护 PMM Server 在 k8s 中各个独立组件，只需维护 pmm-server docker images 容器本身的相关部署即可。

另外，由于阿里云不开放 RDS 实例底层主机管理权限，但又未像 AWS 那样有 API 支持，故阿里云 RDS 无法接入 PMM Client。
但由于对于 PMM 最大的需求是 Query Analytics，实际上如果只需要 Query Analytics 功能以及一些数据库内部监控信息的话，是不需要 PMM Client 的，直接在 PMM Server 界面 remote db 中配置数据库链接信息即可。

对于一些数据库底层比较常见的监控指标，可直接由 exporter 本身提供，然后通过在 grafana dashboard links 来实现和 PMM Server Dashboard 的跳转。
账号体系上，两个 grafana 必须配置相同的账号密码，以便用户只需使用一个账号登录。

```
kubectl create -f kube-aliyun-nas-mount/dev/pmm-storageClass.yaml -n monitoring
kubectl create -f kube-aliyun-nas-mount/dev/pmm-PVC.yaml -n monitoring
kubectl create -f exporter-deploy/db-percona-monitoring-management/pmm-server-deployment.yaml
kubectl create -f exporter-deploy/db-percona-monitoring-management/pmm-server-service.yaml
kubectl create -f kube-prom-ingress/dev/pmm-server-ingress.yaml  -n monitoring
```
关于 pmm-server 容器数据目录 /srv 的挂载，由于容器内有多个进程 user 需要 /src 写权限，如果初次直接挂载使用 root 写入 nas 则会导致其他用户不够权限在挂载目录新建及写入。
第一次启动 pmm-server 时，可先不挂载 /srv，而是映射一个临时目录挂载到 nas，pmm-server 正常启动之后，将容器内的 /srv 目录下所有文件复制到临时目录（nas），在第二次启动时再将 /srv 目录挂载到 nas。
```
[root@pmm-server-74cc748545-nvgzh opt]# cd /srv/
[root@pmm-server-74cc748545-nvgzh srv]# ls
alertmanager  clickhouse  grafana  ia  logs  nginx  pmm-distribution  postgres  prometheus  update  victoriametrics
[root@pmm-server-74cc748545-nvgzh srv]# ls -al
total 80
drwxr-xr-x 1 root     root     4096 Sep 21 10:28 .
drwxr-xr-x 1 root     root     4096 Oct 25 09:47 ..
drwxrwxr-x 1 pmm      pmm      4096 Sep 21 10:25 alertmanager
drwxr-xr-x 1 root     pmm      4096 Oct 25 09:47 clickhouse
drwxrwxr-x 1 grafana  grafana  4096 Oct 25 09:47 grafana
drwxr-xr-x 1 root     root     4096 Sep 21 10:25 ia
drwxrwxr-x 1 pmm      pmm      4096 Oct 25 09:47 logs
drwxr-xr-x 2 root     root     4096 Sep 21 10:27 nginx
-rw-r--r-- 1 root     root        6 Sep 21 10:28 pmm-distribution
drwx------ 1 postgres postgres 4096 Oct 25 09:47 postgres
drwxr-xr-x 1 pmm      pmm      4096 Sep 21 10:24 prometheus
drwxr-xr-x 2 root     root     4096 Sep 21 10:25 update
drwxrwxr-x 1 pmm      pmm      4096 Sep 21 10:26 victoriametrics
```  
如上所示，实际上对于 percona/pmm-server:2 镜像来说，它包括了 postgres、clickhouse、prometheus 等多个有状态的组件。

使用 deployment 部署 pmm-server 的话，在实例切换的时候会导致数据错乱，单节点 dp 可用性要求不高前提下可以先缩减副本为 0 再重新增加副本，相当于同一时间只有一个实例存在。

更优化的方式是使用 StatefulSet 部署 pmm-server，在变更或多实例情况下，StatefulSet 分别挂载在不同的 PVC 相当于双写，实例间数据读写不会相互影响。
StatefulSet 方式部署 pmm-server 不需要提前新建 PVC，而是在 StatefulSet 中配置，其余资源和 dp 部署相同。
```
kubectl create -f kube-aliyun-nas-mount/dev/pmm-storageClass.yaml -n monitoring
kubectl create -f exporter-deploy/db-percona-monitoring-management/pmm-server-statefulset.yaml
kubectl create -f exporter-deploy/db-percona-monitoring-management/pmm-server-service.yaml
kubectl create -f kube-prom-ingress/dev/pmm-server-ingress.yaml  -n monitoring
```

## 数据库接入
PMM Server 接入数据库和普通应用接入方式一样，使用连接地址、用户名及合适权限。需注意数据库安全组的开放，另一个问题是 PMM Server 部署在 K8S 中，pod 应用每次分配的 IP 不是固定的，需和数据库实例安全组开放的网段地址对应，需做好安全控制。
### mysql 
PMM 官方文档中显示需给 mysql user 授予 SUPER、RELOAD 权限，但如只收监控数据和 Query Analytics 的话，只需要拥有 SELECT、 PROCESS、 REPLICATION CLIENT 即可，对应是阿里云只读权限，建议手动授权。
https://help.aliyun.com/document_detail/146395.html?spm=5176.19908259.help.dexternal.7a591450oLMQcJ

此处可直接使用和 mysql-exporter 相同的账号，PMM 权限配置需求和 mysql-exporter 一样。
```
CREATE USER 'monitor'@'%' IDENTIFIED BY 'passwd' WITH MAX_USER_CONNECTIONS 10;
GRANT SELECT, PROCESS, REPLICATION CLIENT ON *.* TO  'monitor'@'%';
```
mysql 实例要确保 PERFORMANCE_SCHEMA 处于开启状态，才能正常收集到 Query Analytics 的内容。
```
SHOW VARIABLES LIKE 'performance_schema';
```
阿里云 RDS 默认关闭 PERFORMANCE_SCHEMA，修改该参数需重启 RDS 实例，需注意实例版本和重启风险。

高可用版本的 RDS 重启会造成30秒左右的连接中断， RDS基础版实例只有一个数据库节点，没有备节点作为热备份，因此当该节点意外宕机或者执行重启实例、变更配置、版本升级等任务时，会出现较长时间的不可用。

PMM MySQL 链接用户新建必须勾选 Use performance schema，否则也会导致无法收集 Query Analytics 数据。

### Mongodb
#### Mongodb profiler 慢查询记录
Mongodb 接入 QAN 需 Mongodb 先开启 profiler 慢查询记录

https://docs.mongodb.com/manual/tutorial/manage-the-database-profiler/

https://studio3t.com/knowledge-base/articles/mongodb-query-performance/

查询 profiler 启用状态
```
db.getProfilingStatus()
/* 1 */
{
    "was" : 0,
    "slowms" : 100,
    "sampleRate" : 1.0
}
```
根据文档描述，当前 Mongodb 没有开启慢查询收集
```
Level 0 – The profiler is off and does not collect any data. This is the default profiler level.
Level 1 – The profiler collects data for operations that take longer than the value of slowms, which you can set.
Level 2 – The profiler collects data for all operations.
```
现将 profiler 收集等级设置为 2
```
db.setProfilingLevel(2)
/* 1 */
{
    "was" : 0,
    "slowms" : 100,
    "sampleRate" : 1.0,
    "ok" : 1.0
}
```
重新查询 profiler 状态
```
db.getProfilingStatus()
/* 1 */
{
    "was" : 2,
    "slowms" : 100,
    "sampleRate" : 1.0
}
```
阿里云实例可直接在参数列表修改，无需重启。

#### Mongodb 用户权限配置
在 admin 库新建访问控制策略
```
db.createRole({
role: "explainRole",
privileges: [{
resource: {
db: "",
collection: ""
},
actions: [
"listIndexes",
"listCollections",
"dbStats",
"dbHash",
"collStats",
"find"
]
}],
roles:[]
})
```
新建 pmm 链接用户，鉴权库为 admin，关联以上访问权限
```
db.getSiblingDB("admin").createUser({
user: "monitor",
pwd: "AZBMwAG8ghJQ6wVg",
roles: [
{ role: "explainRole", db: "admin" },
{ role: "clusterMonitor", db: "admin" },
{ role: "read", db: "local" }
]
})
```
账号测试 profile 查询
```
db.system.profile.find().limit(10).sort( { ts : -1 } ).pretty()
```
PMM Server 中添加 Mongodb 实例，必须 Use QAN MongoDB Profiler 才能收集到 Mongodb Query 信息。
https://www.percona.com/doc/percona-monitoring-and-management/1.x/qan.html#qan-for-mongodb

### Postgres
Postgres 接入 PMM Server QAN 依赖 Postgres 的 pg_stat_statements 模块，以收集服务器所执行的所有 SQL 语句的执行统计信息。
另外，pg_stat_monitor 是 percona 开源的一款 extension，用于监控 postgresql 的性能，但需要另外安装。
该两模块收集数据的权限方式、维度不一样，可按需选择。

https://pgstats.dev/pg_stat_statements

按照文档所示，新增 pg_stat_statements 模块需要重新分配内存，故需重启实例，且 pg_stat_statements_reset 和 pg_stat_statements 不是全局使用的，可以用 CREATE EXTENSION pg_stat_statements 为特定数据库启用。

```
CREATE EXTENSION pg_stat_statements
ERROR:  could not open extension control file "/sdata/usr/local/pgsql/share/extension/pg_stat_statements.control": No such file or directory
SQL state: 58P01
```
