package addons

const CNIYunionTemplate = `
#### CNI plugin ####
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: yunion
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: yunion
subjects:
- kind: ServiceAccount
  name: yunion
  namespace: kube-system
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: yunion
rules:
  - apiGroups:
      - ""
    resources:
      - pods
    verbs:
      - get
  - apiGroups:
      - ""
    resources:
      - nodes
    verbs:
      - list
      - watch
  - apiGroups:
      - ""
    resources:
      - nodes/status
    verbs:
      - patch
---
kind: ConfigMap
apiVersion: v1
metadata:
  name: yunion-cni-config
  namespace: kube-system
data:
  cni-conf.json: |
    {
      "cniVersion": "0.3.1",
      "name": "yunion-cni",
      "type": "yunion-bridge",
      "isDefaultGateway": false,
      "cluster_ip_range": "{{.ClusterCIDR}}",
      "ipam": {
        "type": "yunion-ipam",
        "auth_url": "{{.AuthUrl}}",
        "admin_user": "{{.AdminUser}}",
        "admin_password": "{{.AdminPassword}}",
        "admin_project": "{{.AdminProject}}",
        "timeout": 30,
        "cluster": "{{.Cluster}}",
        "region": "{{.Region}}"
      }
    }
---
kind: DaemonSet
apiVersion: extensions/v1beta1
metadata:
  name: yunion-cni
  namespace: kube-system
  labels:
    k8s-app: yunion-cni
    lxcfs: "false"
spec:
  template:
    metadata:
      labels:
        lxcfs: "false"
        k8s-app: yunion-cni
    spec:
      serviceAccountName: yunion
      hostNetwork: true
      tolerations:
      - operator: Exists
        effect: NoSchedule
      - operator: Exists
        effect: NoExecute
      containers:
        # Runs yunion/cni container on each Kubernetes node.
        # This container installs the Yunion CNI binaries
        # and CNI network config file on each node.
        - name: install-cni
          image: {{.CNIImage}}
          imagePullPolicy: "Always"
          command: ["/install-cni.sh"]
          env:
          # The CNI network config to install on each node.
          - name: CNI_NETWORK_CONFIG
            valueFrom:
              configMapKeyRef:
                name: yunion-cni-config
                key: cni-conf.json
          - name: CNI_CONF_NAME
            value: "10-yunion.conf"
          volumeMounts:
          - mountPath: /host/opt/cni/bin
            name: host-cni-bin
          - mountPath: /host/etc/cni/net.d
            name: host-cni-net
      volumes:
        - name: host-cni-net
          hostPath:
            path: /etc/cni/net.d
        - name: yunion-cni-config
          configMap:
            name: yunion-cni-config
        - name: host-cni-bin
          hostPath:
            path: /opt/cni/bin
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: yunion
  namespace: kube-system
---
`
