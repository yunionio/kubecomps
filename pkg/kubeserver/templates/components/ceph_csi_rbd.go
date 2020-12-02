package components

import "yunion.io/x/kubecomps/pkg/kubeserver/templates"

type CephCSIRBDConfig struct {
	Namespace     string
	ConfigMapName string
	ConfigJSON    string
	// image: registry.cn-beijing.aliyuncs.com/yunionio/csi-provisioner:v1.4.0
	AttacherImage string
	// image: registry.cn-beijing.aliyuncs.com/yunionio/csi-snapshotter:v1.2.2
	ProvisionerImage string
	// image: registry.cn-beijing.aliyuncs.com/yunionio/csi-snapshotter:v1.2.2
	SnapshotterImage string
	// image: registry.cn-beijing.aliyuncs.com/yunionio/cephcsi:v2.0-canary
	CephCSIImage string
	// image: registry.cn-beijing.aliyuncs.com/yunionio/csi-node-driver-registrar:v1.2.0
	RegistrarImage string
	// image: registry.cn-beijing.aliyuncs.com/yunionio/csi-resizer:v0.4.0
	ResizerImage string
}

func (c CephCSIRBDConfig) GenerateYAML() (string, error) {
	return templates.CompileTemplateFromMap(CephCSIRBDTemplate, c)
}

const CephCSIRBDTemplate = `
### csi-nodeplugin-psp.yaml
---
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: rbd-csi-nodeplugin-psp
  namespace: {{.Namespace}}
spec:
  allowPrivilegeEscalation: true
  allowedCapabilities:
    - 'SYS_ADMIN'
  fsGroup:
    rule: RunAsAny
  privileged: true
  hostNetwork: true
  hostPID: true
  runAsUser:
    rule: RunAsAny
  seLinux:
    rule: RunAsAny
  supplementalGroups:
    rule: RunAsAny
  volumes:
    - 'configMap'
    - 'emptyDir'
    - 'projected'
    - 'secret'
    - 'downwardAPI'
    - 'hostPath'
  allowedHostPaths:
    - pathPrefix: '/dev'
      readOnly: false
    - pathPrefix: '/run/mount'
      readOnly: false
    - pathPrefix: '/sys'
      readOnly: false
    - pathPrefix: '/lib/modules'
      readOnly: true
    - pathPrefix: '/var/lib/kubelet/pods'
      readOnly: false
    - pathPrefix: '/var/lib/kubelet/plugins/rbd.csi.ceph.com'
      readOnly: false
    - pathPrefix: '/var/lib/kubelet/plugins_registry'
      readOnly: false
    - pathPrefix: '/var/lib/kubelet/plugins'
      readOnly: false
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: rbd-csi-nodeplugin-psp
  # replace with non-default namespace name
  namespace: {{.Namespace}}
rules:
  - apiGroups: ['policy']
    resources: ['podsecuritypolicies']
    verbs: ['use']
    resourceNames: ['rbd-csi-nodeplugin-psp']
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: rbd-csi-nodeplugin-psp
  # replace with non-default namespace name
  namespace: {{.Namespace}}
subjects:
  - kind: ServiceAccount
    name: rbd-csi-nodeplugin
    # replace with non-default namespace name
    namespace: {{.Namespace}}
roleRef:
  kind: Role
  name: rbd-csi-nodeplugin-psp
  apiGroup: rbac.authorization.k8s.io
---
### csi-nodeplugin-rbac.yaml
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: rbd-csi-nodeplugin
  namespace: {{.Namespace}}
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: rbd-csi-nodeplugin
aggregationRule:
  clusterRoleSelectors:
    - matchLabels:
        rbac.rbd.csi.ceph.com/aggregate-to-rbd-csi-nodeplugin: "true"
rules: []
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: rbd-csi-nodeplugin-rules
  labels:
    rbac.rbd.csi.ceph.com/aggregate-to-rbd-csi-nodeplugin: "true"
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list"]
  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["get", "list", "update"]
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["get", "list"]
  - apiGroups: [""]
    resources: ["persistentvolumes"]
    verbs: ["get", "list", "watch", "update"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["volumeattachments"]
    verbs: ["get", "list", "watch", "update"]
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get", "list"]
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: rbd-csi-nodeplugin
subjects:
  - kind: ServiceAccount
    name: rbd-csi-nodeplugin
    namespace: {{.Namespace}}
roleRef:
  kind: ClusterRole
  name: rbd-csi-nodeplugin
  apiGroup: rbac.authorization.k8s.io
---
### csi-provisioner-psp.yaml
---
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: rbd-csi-provisioner-psp
  namespace: {{.Namespace}}
spec:
  allowPrivilegeEscalation: true
  allowedCapabilities:
    - 'SYS_ADMIN'
  fsGroup:
    rule: RunAsAny
  privileged: true
  runAsUser:
    rule: RunAsAny
  seLinux:
    rule: RunAsAny
  supplementalGroups:
    rule: RunAsAny
  volumes:
    - 'configMap'
    - 'emptyDir'
    - 'projected'
    - 'secret'
    - 'downwardAPI'
    - 'hostPath'
  allowedHostPaths:
    - pathPrefix: '/dev'
      readOnly: false
    - pathPrefix: '/sys'
      readOnly: false
    - pathPrefix: '/lib/modules'
      readOnly: true
---
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  # replace with non-default namespace name
  namespace: {{.Namespace}}
  name: rbd-csi-provisioner-psp
rules:
  - apiGroups: ['policy']
    resources: ['podsecuritypolicies']
    verbs: ['use']
    resourceNames: ['rbd-csi-provisioner-psp']
---
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: rbd-csi-provisioner-psp
  # replace with non-default namespace name
  namespace: {{.Namespace}}
subjects:
  - kind: ServiceAccount
    name: rbd-csi-provisioner
    # replace with non-default namespace name
    namespace: {{.Namespace}}
roleRef:
  kind: Role
  name: rbd-csi-provisioner-psp
  apiGroup: rbac.authorization.k8s.io
---
### csi-provisioner-rbac.yaml
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: rbd-csi-provisioner
  namespace: {{.Namespace}}
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: rbd-external-provisioner-runner
aggregationRule:
  clusterRoleSelectors:
    - matchLabels:
        rbac.rbd.csi.ceph.com/aggregate-to-rbd-external-provisioner-runner: "true"
rules: []
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: rbd-external-provisioner-runner-rules
  labels:
    rbac.rbd.csi.ceph.com/aggregate-to-rbd-external-provisioner-runner: "true"
rules:
  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list"]
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["list", "watch", "create", "update", "patch"]
  - apiGroups: [""]
    resources: ["persistentvolumes"]
    verbs: ["get", "list", "watch", "create", "update", "delete", "patch"]
  - apiGroups: [""]
    resources: ["persistentvolumeclaims"]
    verbs: ["get", "list", "watch", "update"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["storageclasses"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["snapshot.storage.k8s.io"]
    resources: ["volumesnapshots"]
    verbs: ["get", "list", "watch", "update"]
  - apiGroups: ["snapshot.storage.k8s.io"]
    resources: ["volumesnapshotcontents"]
    verbs: ["create", "get", "list", "watch", "update", "delete"]
  - apiGroups: ["snapshot.storage.k8s.io"]
    resources: ["volumesnapshotclasses"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["apiextensions.k8s.io"]
    resources: ["customresourcedefinitions"]
    verbs: ["create", "list", "watch", "delete", "get", "update"]
  - apiGroups: ["csi.storage.k8s.io"]
    resources: ["csinodeinfos"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["volumeattachments"]
    verbs: ["get", "list", "watch", "update", "patch"]
  - apiGroups: ["snapshot.storage.k8s.io"]
    resources: ["volumesnapshots/status"]
    verbs: ["update"]
  - apiGroups: [""]
    resources: ["persistentvolumeclaims/status"]
    verbs: ["update", "patch"]

---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: rbd-csi-provisioner-role
subjects:
  - kind: ServiceAccount
    name: rbd-csi-provisioner
    namespace: {{.Namespace}}
roleRef:
  kind: ClusterRole
  name: rbd-external-provisioner-runner
  apiGroup: rbac.authorization.k8s.io
---
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  # replace with non-default namespace name
  namespace: {{.Namespace}}
  name: rbd-external-provisioner-cfg
rules:
  - apiGroups: [""]
    resources: ["endpoints"]
    verbs: ["get", "watch", "list", "delete", "update", "create"]
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get", "list", "watch", "create", "delete"]
  - apiGroups: ["coordination.k8s.io"]
    resources: ["leases"]
    verbs: ["get", "watch", "list", "delete", "update", "create"]

---
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: rbd-csi-provisioner-role-cfg
  # replace with non-default namespace name
  namespace: {{.Namespace}}
subjects:
  - kind: ServiceAccount
    name: rbd-csi-provisioner
    # replace with non-default namespace name
    namespace: {{.Namespace}}
roleRef:
  kind: Role
  name: rbd-external-provisioner-cfg
  apiGroup: rbac.authorization.k8s.io
---
### csi-rbdplugin-provisioner.yaml
---
kind: Service
apiVersion: v1
metadata:
  name: csi-rbdplugin-provisioner
  namespace: {{.Namespace}}
  labels:
    app: csi-metrics
spec:
  selector:
    app: csi-rbdplugin-provisioner
  ports:
    - name: http-metrics
      port: 8080
      protocol: TCP
      targetPort: 8680
    - name: grpc-metrics
      port: 8090
      protocol: TCP
      targetPort: 8090
---
kind: Deployment
apiVersion: apps/v1
metadata:
  name: csi-rbdplugin-provisioner
  namespace: {{.Namespace}}
spec:
  replicas: 3
  selector:
    matchLabels:
      app: csi-rbdplugin-provisioner
  template:
    metadata:
      labels:
        app: csi-rbdplugin-provisioner
    spec:
      serviceAccount: rbd-csi-provisioner
      containers:
        - name: csi-provisioner
          # image: registry.cn-beijing.aliyuncs.com/yunionio/csi-provisioner:v1.4.0
          image: {{.ProvisionerImage}}
          args:
            - "--csi-address=$(ADDRESS)"
            - "--v=5"
            - "--timeout=150s"
            - "--retry-interval-start=500ms"
            - "--enable-leader-election=true"
            - "--leader-election-type=leases"
          env:
            - name: ADDRESS
              value: unix:///csi/csi-provisioner.sock
          imagePullPolicy: "IfNotPresent"
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
        - name: csi-snapshotter
          # image: registry.cn-beijing.aliyuncs.com/yunionio/csi-snapshotter:v1.2.2
          image: {{.SnapshotterImage}}
          args:
            - "--csi-address=$(ADDRESS)"
            - "--v=5"
            - "--timeout=150s"
            - "--leader-election=true"
          env:
            - name: ADDRESS
              value: unix:///csi/csi-provisioner.sock
          imagePullPolicy: Always
          securityContext:
            privileged: true
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
        - name: csi-attacher
          # image: registry.cn-beijing.aliyuncs.com/yunionio/csi-attacher:v2.1.0
          image: {{.AttacherImage}}
          args:
            - "--v=5"
            - "--csi-address=$(ADDRESS)"
            - "--leader-election=true"
          env:
            - name: ADDRESS
              value: /csi/csi-provisioner.sock
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
          imagePullPolicy: "IfNotPresent"
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
        - name: csi-resizer
          # image: registry.cn-beijing.aliyuncs.com/yunionio/csi-resizer:v0.4.0
          image: {{.ResizerImage}}
          args:
            - "--csi-address=$(ADDRESS)"
            - "--v=5"
            - "--csiTimeout=150s"
            - "--leader-election"
          env:
            - name: ADDRESS
              value: unix:///csi/csi-provisioner.sock
          imagePullPolicy: "IfNotPresent"
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
        - name: csi-rbdplugin
          securityContext:
            privileged: true
            capabilities:
              add: ["SYS_ADMIN"]
          # for stable functionality replace canary with latest release version
          # image: registry.cn-beijing.aliyuncs.com/yunionio/cephcsi:v2.0-canary
          image: {{.CephCSIImage}}
          args:
            - "--nodeid=$(NODE_ID)"
            - "--type=rbd"
            - "--controllerserver=true"
            - "--endpoint=$(CSI_ENDPOINT)"
            - "--v=5"
            - "--drivername=rbd.csi.ceph.com"
            - "--pidlimit=-1"
            - "--metricsport=8090"
            - "--metricspath=/metrics"
            - "--enablegrpcmetrics=false"
          env:
            - name: POD_IP
              valueFrom:
                fieldRef:
                  fieldPath: status.podIP
            - name: NODE_ID
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: CSI_ENDPOINT
              value: unix:///csi/csi-provisioner.sock
          imagePullPolicy: "IfNotPresent"
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
            - mountPath: /dev
              name: host-dev
            - mountPath: /sys
              name: host-sys
            - mountPath: /lib/modules
              name: lib-modules
              readOnly: true
            - name: {{.ConfigMapName}}
              mountPath: /etc/ceph-csi-config/
            - name: keys-tmp-dir
              mountPath: /tmp/csi/keys
        - name: liveness-prometheus
          #image: registry.cn-beijing.aliyuncs.com/yunionio/cephcsi:v2.0-canary
          image: {{.CephCSIImage}}
          args:
            - "--type=liveness"
            - "--endpoint=$(CSI_ENDPOINT)"
            - "--metricsport=8680"
            - "--metricspath=/metrics"
            - "--polltime=60s"
            - "--timeout=3s"
          env:
            - name: CSI_ENDPOINT
              value: unix:///csi/csi-provisioner.sock
            - name: POD_IP
              valueFrom:
                fieldRef:
                  fieldPath: status.podIP
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
          imagePullPolicy: "IfNotPresent"
      volumes:
        - name: host-dev
          hostPath:
            path: /dev
        - name: host-sys
          hostPath:
            path: /sys
        - name: lib-modules
          hostPath:
            path: /lib/modules
        - name: socket-dir
          emptyDir: {
            medium: "Memory"
          }
        - name: ceph-csi-config
          configMap:
            name: ceph-csi-config
        - name: keys-tmp-dir
          emptyDir: {
            medium: "Memory"
          }
---
### csi-rbdplugin.yaml
---
kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: csi-rbdplugin
  namespace: {{.Namespace}}
spec:
  selector:
    matchLabels:
      app: csi-rbdplugin
  template:
    metadata:
      labels:
        app: csi-rbdplugin
    spec:
      serviceAccount: rbd-csi-nodeplugin
      hostNetwork: true
      hostPID: true
      # to use e.g. Rook orchestrated cluster, and mons' FQDN is
      # resolved through k8s service, set dns policy to cluster first
      dnsPolicy: ClusterFirstWithHostNet
      containers:
        - name: driver-registrar
          # This is necessary only for systems with SELinux, where
          # non-privileged sidecar containers cannot access unix domain socket
          # created by privileged CSI driver container.
          securityContext:
            privileged: true
          #image: registry.cn-beijing.aliyuncs.com/yunionio/csi-node-driver-registrar:v1.2.0
          image: {{.RegistrarImage}}
          args:
            - "--v=5"
            - "--csi-address=/csi/csi.sock"
            - "--kubelet-registration-path=/var/lib/kubelet/plugins/rbd.csi.ceph.com/csi.sock"
          lifecycle:
            preStop:
              exec:
                command: [
                  "/bin/sh", "-c",
                  "rm -rf /registration/rbd.csi.ceph.com \
                  /registration/rbd.csi.ceph.com-reg.sock"
                ]
          env:
            - name: KUBE_NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
            - name: registration-dir
              mountPath: /registration
        - name: csi-rbdplugin
          securityContext:
            privileged: true
            capabilities:
              add: ["SYS_ADMIN"]
            allowPrivilegeEscalation: true
          # for stable functionality replace canary with latest release version
          image: {{.CephCSIImage}}
          args:
            - "--nodeid=$(NODE_ID)"
            - "--type=rbd"
            - "--nodeserver=true"
            - "--endpoint=$(CSI_ENDPOINT)"
            - "--v=5"
            - "--drivername=rbd.csi.ceph.com"
            - "--metricsport=8090"
            - "--metricspath=/metrics"
            - "--enablegrpcmetrics=false"
          env:
            - name: POD_IP
              valueFrom:
                fieldRef:
                  fieldPath: status.podIP
            - name: NODE_ID
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: CSI_ENDPOINT
              value: unix:///csi/csi.sock
          imagePullPolicy: "IfNotPresent"
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
            - mountPath: /dev
              name: host-dev
            - mountPath: /sys
              name: host-sys
            - mountPath: /run/mount
              name: host-mount
            - mountPath: /lib/modules
              name: lib-modules
              readOnly: true
            - name: ceph-csi-config
              mountPath: /etc/ceph-csi-config/
            - name: plugin-dir
              mountPath: /var/lib/kubelet/plugins
              mountPropagation: "Bidirectional"
            - name: mountpoint-dir
              mountPath: /var/lib/kubelet/pods
              mountPropagation: "Bidirectional"
            - name: keys-tmp-dir
              mountPath: /tmp/csi/keys
        - name: liveness-prometheus
          securityContext:
            privileged: true
          image: {{.CephCSIImage}}
          args:
            - "--type=liveness"
            - "--endpoint=$(CSI_ENDPOINT)"
            - "--metricsport=8680"
            - "--metricspath=/metrics"
            - "--polltime=60s"
            - "--timeout=3s"
          env:
            - name: CSI_ENDPOINT
              value: unix:///csi/csi.sock
            - name: POD_IP
              valueFrom:
                fieldRef:
                  fieldPath: status.podIP
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
          imagePullPolicy: "IfNotPresent"
      volumes:
        - name: socket-dir
          hostPath:
            path: /var/lib/kubelet/plugins/rbd.csi.ceph.com
            type: DirectoryOrCreate
        - name: plugin-dir
          hostPath:
            path: /var/lib/kubelet/plugins
            type: Directory
        - name: mountpoint-dir
          hostPath:
            path: /var/lib/kubelet/pods
            type: DirectoryOrCreate
        - name: registration-dir
          hostPath:
            path: /var/lib/kubelet/plugins_registry/
            type: Directory
        - name: host-dev
          hostPath:
            path: /dev
        - name: host-sys
          hostPath:
            path: /sys
        - name: host-mount
          hostPath:
            path: /run/mount
        - name: lib-modules
          hostPath:
            path: /lib/modules
        - name: ceph-csi-config
          configMap:
            name: ceph-csi-config
        - name: keys-tmp-dir
          emptyDir: {
            medium: "Memory"
          }
---
# This is a service to expose the liveness and grpc metrics
apiVersion: v1
kind: Service
metadata:
  name: csi-metrics-rbdplugin
  namespace: {{.Namespace}}
  labels:
    app: csi-metrics
spec:
  ports:
    - name: http-metrics
      port: 8080
      protocol: TCP
      targetPort: 8680
    - name: grpc-metrics
      port: 8090
      protocol: TCP
      targetPort: 8090
  selector:
    app: csi-rbdplugin
---
### ceph-csi-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{.ConfigMapName}}
  namespace: {{.Namespace}}
data:
  config.json: |-
    {{.ConfigJSON}}
---
`
