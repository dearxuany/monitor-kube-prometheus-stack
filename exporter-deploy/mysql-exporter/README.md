# Mysql Prometheus Exporter
mysql metrics 数据流向与 nginx-exporter 类似
```
mysql -> mysql-exporter dp -> service -> servicemonitor -> prometheus -> grafana
```
https://github.com/prometheus/mysqld_exporter
## mysql 账号配置
需要 5.6 版本以上 mysql 才支持 mysql-exporter metrics 数据抓取
```
select version();
```
mysql 创建 exporter 查询账号
```
CREATE USER 'exporter'@'ip' IDENTIFIED BY 'XXXXXXXX' WITH MAX_USER_CONNECTIONS 3;
GRANT PROCESS, REPLICATION CLIENT, SELECT ON *.* TO 'exporter'@'ip';
```
需调整可用链接数及访问 IP 网段限制，注意 mysql-exporter 部署在 k8s 中， pod IP 并不是固定的。
```
CREATE USER 'monitor'@'10.0.0.%' IDENTIFIED BY 'XXXXXX' WITH MAX_USER_CONNECTIONS 3;
GRANT PROCESS, REPLICATION CLIENT, SELECT ON *.* TO 'monitor'@'10.0.0.%';
```
对于已存在的账号，直接修改权限，不同版本的 mysql 修改命令可能不同，需对照官方文档。
https://dev.mysql.com/doc/refman/8.0/en/user-resources.html 
```
# mysql 5.6
GRANT PROCESS, REPLICATION CLIENT, SELECT ON *.* TO 'monitor'@'%' WITH MAX_USER_CONNECTIONS 6;

# mysql >= 5.7
GRANT PROCESS, REPLICATION CLIENT, SELECT ON *.* TO 'monitor'@'10.0.0.%';
ALTER USER 'monitor'@'10.0.0.%' WITH MAX_USER_CONNECTIONS 3;
```
查询授权信息
```
SELECT user,host FROM mysql.user;
SELECT * FROM mysql.user WHERE user='monitor'
SHOW GRANTS FOR 'monitor'@'%';
```
注意：如果是阿里云读写分离实例，注意账号授权的时候 MAX_USER_CONNECTIONS 需至少调为 6。

## mysql-exporter 部署
与 nginx-exporter 不同的是，mysql-exporter 需要 mysql 的账号密码。
此处出于安全考虑，将新建 secret 类型资源用于存储 mysql 账号密码。
```
# kubectl create -f ./mysql-exporter/dev/ --kubeconfig ~/.kubeconfig/dev/dev-admin.kubeconfig 
deployment.apps/mysql-exporter created
secret/mysql-exporter-secret created
service/mysql-exporter created
servicemonitor.monitoring.coreos.com/mysql-exporter created
```
prometheus target
注意：多个不同实例的 mysql deployment name 必须不同，但 service 及 servicemonitor 用一个即可，使用 k8s-app label 关联。
```
monitoring/mysql-exporter/0 (1/1 up) 
Endpoint	State	Labels	Last Scrape	Scrape Duration	Error
http://10.42.11.161:9104/metrics
UP	container="mysql-exporter" endpoint="mysql-exporter" instance="gzyw53-dev-mysql-01" job="mysql-exporter" namespace="monitoring" pod="gzyw53-dev-mysql-01-metrics-c65d8c587-lt6v4" service="mysql-exporter"
```
跨 ns 访问 mysql-exporter metrics
```
# curl http://mysql-exporter.monitoring.svc.cluster.local:9104/metrics
mysql_up 1
# HELP mysql_version_info MySQL version and distribution.
# TYPE mysql_version_info gauge
mysql_version_info{innodb_version="5.7.29",version="5.7.29-log",version_comment="MySQL Community Server (GPL)"} 1
# HELP mysqld_exporter_build_info A metric with a constant '1' value labeled by version, revision, branch, and goversion from which mysqld_exporter was built.
# TYPE mysqld_exporter_build_info gauge
mysqld_exporter_build_info{branch="HEAD",goversion="go1.16.4",revision="ad2847c7fa67b9debafccd5a08bacb12fc9031f1",version="0.13.0"} 1
# HELP process_cpu_seconds_total Total user and system CPU time spent in seconds.
# TYPE process_cpu_seconds_total counter
process_cpu_seconds_total 508.86
# HELP process_max_fds Maximum number of open file descriptors.
# TYPE process_max_fds gauge
process_max_fds 1.048576e+06
# HELP process_open_fds Number of open file descriptors.
# TYPE process_open_fds gauge
process_open_fds 11
# HELP process_resident_memory_bytes Resident memory size in bytes.
# TYPE process_resident_memory_bytes gauge
process_resident_memory_bytes 1.4200832e+07
```
## metrics 数据收集粒度
mysql-exporter 支持根据 args 启动参数形式设置 mertrics 数据的收集粒度，比如收集一些表级别的监控数据，可以按需开启。

不同版本的 mysql 会有不同配置的 collect 参数，需根据 mysql 版本来选择具体的参数。
```
# 部分通用参数
      - args:
        - --collect.binlog_size
        - --collect.engine_innodb_status
        - --collect.global_status
        - --collect.global_variables
        - --collect.info_schema.processlist
        - --collect.info_schema.tables
        - --collect.info_schema.tablestats
        - --collect.info_schema.schemastats
        - --collect.info_schema.userstats
        - --collect.slave_status
        - --collect.slave_hosts

# mysql 5.6 可启用参数
      - args:
        - --collect.auto_increment.columns
        - --collect.engine_tokudb_status
        - --collect.info_schema.innodb_metrics
        - --collect.info_schema.replica_host
        - --collect.perf_schema.eventsstatements
        - --collect.perf_schema.file_events
        - --collect.perf_schema.indexiowaits
        - --collect.perf_schema.tableiowaits
        - --collect.perf_schema.tablelocks


# mysql 5.7 可启用参数
      - args:
        - --collect.auto_increment.columns
        - --collect.info_schema.innodb_tablespaces
        - --collect.perf_schema.eventsstatementssum
        - --collect.perf_schema.memory_events
        - --collect.perf_schema.replication_group_members
        - --collect.perf_schema.replication_group_member_stats
```

开启 collect 参数后，会一定程度增加 mysql-exporter 对 mysql 访问的链接数，故需适当增加 MAX_USER_CONNECTIONS。
```
GRANT USAGE ON *.* TO 'monitor'@'10.0.0.%' WITH MAX_USER_CONNECTIONS 20;
```
对于和 mysql 主从相关的一些监控参数，需要 mysql user 拥有 REPLICATION SLAVE 权限。
```
GRANT PROCESS, REPLICATION CLIENT, REPLICATION SLAVE, SELECT ON *.* TO 'monitor'@'10.0.0.%';
```