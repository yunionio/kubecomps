package addons

const CSIYunionTemplate = `
#### controller parts ####

apiVersion: v1
kind: ServiceAccount
metadata:
  name: csi-yunion-controller-sa
  namespace: kube-system

---
# external attacher
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: csi-attacher-role
rules:
  - apiGroups: [""]
    resources: ["persistentvolumes"]
    verbs: ["get", "list", "watch", "update"]
  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["volumeattachments"]
    verbs: ["get", "list", "watch", "update"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["csinodes"]
    verbs: ["get", "list", "watch"]


---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: csi-attacher-binding
subjects:
  - kind: ServiceAccount
    name: csi-yunion-controller-sa
    namespace: kube-system
roleRef:
  kind: ClusterRole
  name: csi-attacher-role
  apiGroup: rbac.authorization.k8s.io

---
# external Provisioner
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: csi-provisioner-role
rules:
  - apiGroups: [""]
    resources: ["persistentvolumes"]
    verbs: ["get", "list", "watch", "create", "delete"]
  - apiGroups: [""]
    resources: ["persistentvolumeclaims"]
    verbs: ["get", "list", "watch", "update"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["storageclasses"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["csinodes"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["list", "watch", "create", "update", "patch"]
  - apiGroups: ["snapshot.storage.k8s.io"]
    resources: ["volumesnapshots"]
    verbs: ["get", "list"]
  - apiGroups: ["snapshot.storage.k8s.io"]
    resources: ["volumesnapshotcontents"]
    verbs: ["get", "list"]

---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: csi-provisioner-binding
subjects:
  - kind: ServiceAccount
    name: csi-yunion-controller-sa
    namespace: kube-system
roleRef:
  kind: ClusterRole
  name: csi-provisioner-role
  apiGroup: rbac.authorization.k8s.io

---
# external snapshotter
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: csi-snapshotter-role
rules:
  - apiGroups: [""]
    resources: ["persistentvolumes"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["persistentvolumeclaims"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["storageclasses"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["list", "watch", "create", "update", "patch"]
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list"]
  - apiGroups: ["snapshot.storage.k8s.io"]
    resources: ["volumesnapshotclasses"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["snapshot.storage.k8s.io"]
    resources: ["volumesnapshotcontents"]
    verbs: ["create", "get", "list", "watch", "update", "delete"]
  - apiGroups: ["snapshot.storage.k8s.io"]
    resources: ["volumesnapshots"]
    verbs: ["get", "list", "watch", "update"]
  - apiGroups: ["apiextensions.k8s.io"]
    resources: ["customresourcedefinitions"]
    verbs: ["create", "list", "watch", "delete"]

---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: csi-snapshotter-binding
subjects:
  - kind: ServiceAccount
    name: csi-yunion-controller-sa
    namespace: kube-system
roleRef:
  kind: ClusterRole
  name: csi-snapshotter-role
  apiGroup: rbac.authorization.k8s.io

---
kind: Service
apiVersion: v1
metadata:
  name: csi-yunion-controller-service
  namespace: kube-system
  labels:
    app: csi-yunion-controllerplugin
spec:
  selector:
    app: csi-yunion-controllerplugin
  ports:
    - name: dummy
      port: 12345

---
kind: StatefulSet
apiVersion: apps/v1
metadata:
  name: csi-yunion-controllerplugin
  namespace: kube-system
spec:
  serviceName: "csi-yunion-controller-service"
  replicas: 1
  selector:
    matchLabels:
      app: csi-yunion-controllerplugin
  template:
    metadata:
      labels:
        app: csi-yunion-controllerplugin
    spec:
      tolerations:
      - effect: NoSchedule
        operator: Exists
        key: node-role.kubernetes.io/master
      - effect: NoSchedule
        operator: Exists
        key: node.cloudprovider.kubernetes.io/uninitialized
      serviceAccount: csi-yunion-controller-sa
      containers:
        - name: csi-attacher
          image: {{.AttacherImage}}
          args:
            - "--v=5"
            - "--csi-address=$(ADDRESS)"
          env:
            - name: ADDRESS
              value: /csi/csi.sock
          imagePullPolicy: "IfNotPresent"
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
        - name: csi-provisioner
          image: {{.ProvisionerImage}}
          args:
            - "--provisioner=disk.csi.yunion.io"
            - "--csi-address=$(ADDRESS)"
            - "--v=5"
          env:
            - name: ADDRESS
              value: /csi/csi.sock
          imagePullPolicy: "IfNotPresent"
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
        - name: yunion-csi-plugin
          image: {{.PluginImage}}
          command:
            - "/bin/yunion-csi-plugin"
          args :
            - "--nodeid=$(NODE_ID)"
            - "--endpoint=$(CSI_ENDPOINT)"
            - "--cloud-config=$(CLOUD_CONFIG)"
            - "--run-as-controller"
          env:
            - name: NODE_ID
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: CSI_ENDPOINT
              value: unix:///csi/csi.sock
            - name: CLOUD_CONFIG
              value: /etc/config/cloud.conf
          imagePullPolicy: "Always"
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
            - name: secret-csiplugin
              mountPath: /etc/config
              readOnly: true
      volumes:
        - name: socket-dir
          emptyDir:
          #hostPath:
            #path: /var/lib/kubelet/plugins/disk.csi.yunion.io
            #type: DirectoryOrCreate
        - name: secret-csiplugin
          secret:
            secretName: cloud-config
---

#### node parts ####
apiVersion: v1
kind: ServiceAccount
metadata:
  name: csi-yunion-node-sa
  namespace: kube-system
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: csi-nodeplugin-role
rules:
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["get", "list", "watch", "create", "update", "patch"]
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
  - apiGroups: ["csi.storage.k8s.io"]
    resources: ["csidrivers"]
    verbs: ["get", "list", "watch", "update"]
  - apiGroups: ["csi.storage.k8s.io"]
    resources: ["csinodeinfos"]
    verbs: ["get", "list", "watch", "update"]

---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: csi-nodeplugin-binding
subjects:
  - kind: ServiceAccount
    name: csi-yunion-node-sa
    namespace: kube-system
roleRef:
  kind: ClusterRole
  name: csi-nodeplugin-role
  apiGroup: rbac.authorization.k8s.io

---
kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: csi-yunion-nodeplugin
  namespace: kube-system
spec:
  selector:
    matchLabels:
      app: csi-yunion-nodeplugin
  template:
    metadata:
      labels:
        app: csi-yunion-nodeplugin
    spec:
      serviceAccount: csi-yunion-node-sa
      hostNetwork: true
      hostPID: true
      containers:
        - name: node-driver-registrar
          image: {{.RegistrarImage}}
          args:
            - "--v=5"
            - "--csi-address=$(ADDRESS)"
            - "--kubelet-registration-path=$(DRIVER_REG_SOCK_PATH)"
          lifecycle:
            preStop:
              exec:
                command: ["/bin/sh", "-c", "rm -rf /registration/disk.csi.yunion.io /registration/disk.csi.yunion.io-reg.sock"]
          env:
            - name: ADDRESS
              value: /csi/csi.sock
            - name: DRIVER_REG_SOCK_PATH
              value: /var/lib/kubelet/plugins/disk.csi.yunion.io/csi.sock
            - name: KUBE_NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
          imagePullPolicy: "IfNotPresent"
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
            - name: registration-dir
              mountPath: /registration
        - name: yunion-csi-plugin
          securityContext:
            privileged: true
            capabilities:
              add: ["SYS_ADMIN"]
            allowPrivilegeEscalation: true
          image: {{.PluginImage}}
          command:
            - "/bin/yunion-csi-plugin"
          args :
            - "--nodeid=$(NODE_ID)"
            - "--endpoint=$(CSI_ENDPOINT)"
            - "--cloud-config=$(CLOUD_CONFIG)"
          env:
            - name: NODE_ID
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: CSI_ENDPOINT
              value: unix://csi/csi.sock
            - name: CLOUD_CONFIG
              value: /etc/config/cloud.conf
          imagePullPolicy: "Always"
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
            - name: pods-mount-dir
              mountPath: /var/lib/kubelet/pods
              mountPropagation: "Bidirectional"
            - name: pods-probe-dir
              mountPath: /dev
              mountPropagation: "HostToContainer"
            - name: secret-csiplugin
              mountPath: /etc/config
              readOnly: true
            - mountPath: /lib/modules
              name: lib-modules
              readOnly: true
            - mountPath: /opt/cloud/workspace
              name: yunion-workspace
      volumes:
        - name: socket-dir
          hostPath:
            path: /var/lib/kubelet/plugins/disk.csi.yunion.io
            type: DirectoryOrCreate
        - name: registration-dir
          hostPath:
            path: /var/lib/kubelet/plugins_registry/
            type: Directory
        - name: pods-mount-dir
          hostPath:
            path: /var/lib/kubelet/pods
            type: Directory
        - name: pods-probe-dir
          hostPath:
            path: /dev
            type: Directory
        - name: lib-modules
          hostPath:
            path: /lib/modules
        - name: yunion-workspace
          hostPath:
            path: /opt/cloud/workspace
            type: DirectoryOrCreate
        - name: secret-csiplugin
          secret:
            secretName: cloud-config

---
#### secret config parts ####
kind: Secret
apiVersion: v1
metadata:
  name: cloud-config
  namespace: kube-system
data:
  cloud.conf: {{.Base64Config}}

---
#### storageclass ####
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: csi-yunion
  annotations:
    storageclass.kubernetes.io/is-default-class: "true"
provisioner: disk.csi.yunion.io
volumeBindingMode: WaitForFirstConsumer
reclaimPolicy: Delete
`
