# kubernetes 安装阿里云 Nas Kubernetes CSI 插件
NAS CSI插件支持为应用负载挂载阿里云NAS存储卷，也支持动态创建NAS卷。NAS存储是一种共享存储，可以同时被多个应用负载使用(ReadWriteMany)。

https://github.com/kubernetes-sigs/alibaba-cloud-csi-driver.git

如下操作完毕后即可，部署 k8s 应用的 storageClass、PV、PVC
```
# kubectl create -f alibaba-cloud-csi-driver-master/deploy/rbac.yaml
# kubectl create -f alibaba-cloud-csi-driver-master/deploy/nas/nas-plugin.yaml
# kubectl create -f alibaba-cloud-csi-driver-master/deploy/nas/nas-provisioner.yaml
```
非阿里云 k8s 托管集群会报 nodeID 的错，导致 nas-plugin 无法成功启动
```
Running nas plugin....
2021/8/24 下午4:15:24 time="2021-08-24T16:15:24+08:00" level=info msg="Multi CSI Driver Name: nas, nodeID: , endPoints: unix://var/lib/kubelet/csi-plugins/driverplugin.csi.alibabacloud.com-replace/csi.sock"
2021/8/24 下午4:15:24 time="2021-08-24T16:15:24+08:00" level=info msg="CSI Driver Branch: 'master', Version: 'v1.18.8.47-906bd535-aliyun', Build time: '2021-05-13-20:56:55'\n"
2021/8/24 下午4:15:24 time="2021-08-24T16:15:24+08:00" level=info msg="Create Stroage Path: /var/lib/kubelet/csi-plugins/nasplugin.csi.alibabacloud.com/controller"
2021/8/24 下午4:15:24 time="2021-08-24T16:15:24+08:00" level=info msg="Create Stroage Path: /var/lib/kubelet/csi-plugins/nasplugin.csi.alibabacloud.com/node"
2021/8/24 下午4:15:24 time="2021-08-24T16:15:24+08:00" level=info msg="CSI is running status."
2021/8/24 下午4:15:24 time="2021-08-24T16:15:24+08:00" level=info msg="Metric listening on address: /healthz"
2021/8/24 下午4:15:24 time="2021-08-24T16:15:24+08:00" level=info msg="Driver: nasplugin.csi.alibabacloud.com version: 1.0.0"
2021/8/24 下午4:15:24 time="2021-08-24T16:15:24+08:00" level=info msg="Metric listening on address: /metrics"
2021/8/24 下午4:15:54 E0824 16:15:54.484098 2706 driver.go:46] NodeID missing
2021/8/24 下午4:15:54 I0824 16:15:54.484126 2706 driver.go:93] Enabling volume access mode: MULTI_NODE_MULTI_WRITER
2021/8/24 下午4:15:54 time="2021-08-24T16:15:54+08:00" level=info msg="Use node id : "
2021/8/24 下午4:15:54 panic: runtime error: invalid memory address or nil pointer dereference
2021/8/24 下午4:15:54 [signal SIGSEGV: segmentation violation code=0x1 addr=0x50 pc=0x17cac64]
2021/8/24 下午4:15:54
2021/8/24 下午4:15:54 goroutine 13 [running]:
2021/8/24 下午4:15:54 github.com/kubernetes-csi/drivers/pkg/csi-common.(*CSIDriver).AddVolumeCapabilityAccessModes(0x0, 0xc00010feec, 0x1, 0x1, 0x0, 0x0, 0x0)
2021/8/24 下午4:15:54 /home/regressionTest/go/pkg/mod/github.com/kubernetes-csi/drivers@v1.0.2/pkg/csi-common/driver.go:96 +0x1c4
2021/8/24 下午4:15:54 github.com/kubernetes-sigs/alibaba-cloud-csi-driver/pkg/nas.NewDriver(0x0, 0x0, 0xc00034c460, 0x4a, 0x0)
2021/8/24 下午4:15:54 /home/regressionTest/go/src/github.com/kubernetes-sigs/alibaba-cloud-csi-driver/pkg/nas/nas.go:86 +0x1b5
2021/8/24 下午4:15:54 main.main.func1(0xc000407ec0, 0xc00034c460, 0x4a)
2021/8/24 下午4:15:54 /home/regressionTest/go/src/github.com/kubernetes-sigs/alibaba-cloud-csi-driver/main.go:180 +0x79
2021/8/24 下午4:15:54 created by main.main
2021/8/24 下午4:15:54 /home/regressionTest/go/src/github.com/kubernetes-sigs/alibaba-cloud-csi-driver/main.go:178 +0xfe8
```
使用节点名称做 nodeID，其中 ${KUBE_NODE_NAME} 为 k8s pod 环境变量

https://github.com/kubernetes-sigs/alibaba-cloud-csi-driver/issues/400

```
# vim alibaba-cloud-csi-driver-master/deploy/nas/nas-plugin.yaml

         image: registry.cn-hangzhou.aliyuncs.com/acs/csi-plugin:v1.18.8.47-906bd535-aliyun
          imagePullPolicy: "Always"
          args:
            - "--endpoint=$(CSI_ENDPOINT)"
            - "--v=2"
            - "--driver=nas"
            - "--nodeid=${KUBE_NODE_NAME}"
          env:
            - name: KUBE_NODE_NAME
              valueFrom:
                fieldRef:
                  apiVersion: v1
                  fieldPath: spec.nodeName


# vim alibaba-cloud-csi-driver-master/deploy/nas/nas-provisioner.yaml

        - name: csi-provisioner
          securityContext:
            privileged: true
            capabilities:
              add: ["SYS_ADMIN"]
            allowPrivilegeEscalation: true
          image: registry.cn-hangzhou.aliyuncs.com/acs/csi-plugin:v1.18.8.47-906bd535-aliyun
          imagePullPolicy: "Always"
          args:
            - "--endpoint=$(CSI_ENDPOINT)"
            - "--v=2"
            - "--driver=nas"
            - "--nodeid=${KUBE_NODE_NAME}"
          env:
            - name: CSI_ENDPOINT
              value: unix://var/lib/kubelet/csi-provisioner/driverplugin.csi.alibabacloud.com-replace/csi.sock
            - name: MAX_VOLUMES_PERNODE
              value: "15"
            - name: SERVICE_TYPE
              value: "provisioner"
            - name: KUBE_NODE_NAME
              valueFrom:
                fieldRef:
                  apiVersion: v1
                  fieldPath: spec.nodeName
```