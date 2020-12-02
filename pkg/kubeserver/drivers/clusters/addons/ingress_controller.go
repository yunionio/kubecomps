package addons

const (
	YunionIngressControllerTemplate string = `
####### ingress controller ######
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ingress-controller-manager
  namespace: kube-system
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: system:ingress-controller-manager
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: ingress-controller-manager
  namespace: kube-system
---
kind: ConfigMap
apiVersion: v1
metadata:
  labels:
    k8s-app: yunion-ingress-controller-manager
  name: yunion-ingress-controller-manager-config
  namespace: kube-system
data:
  cloud-config.json: |
    {
      "auth_url": "{{.AuthUrl}}",
      "admin_user": "{{.AdminUser}}",
      "admin_password": "{{.AdminPassword}}",
      "admin_project": "{{.AdminProject}}",
      "region": "{{.Region}}",
      "cluster": "{{.Cluster}}",
      "instance_type": "{{.InstanceType}}"
    }
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    k8s-app: yunion-ingress-controller-manager
  annotations:
     scheduler.alpha.kubernetes.io/critical-pod: ''
  name: yunion-ingress-controller-manager
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      k8s-app: yunion-ingress-controller-manager
  template:
    metadata:
      labels:
        k8s-app: yunion-ingress-controller-manager
    spec:
      hostNetwork: true
      serviceAccountName: ingress-controller-manager
      tolerations:
      # this is required so CCM can bootstrap itself
      - key: node.cloudprovider.kubernetes.io/uninitialized
        value: "true"
        effect: NoSchedule
      # cloud controller manages should be able to run on masters
      - key: node-role.kubernetes.io/master
        effect: NoSchedule
      - key: node-role.kubernetes.io/controlplane
        effect: NoSchedule
      # this is to restrict CCM to only run on master nodes
      # the node selector may vary depending on your cluster setup
      containers:
      - name: ingress-controller-manager
        image: {{.Image}}
        command:
        - /ingress-controller
        - --config=/etc/kubernetes/cloud-config.json
        volumeMounts:
        - mountPath: /etc/kubernetes
          name: kubernetes-config
      volumes:
      - configMap:
          defaultMode: 420
          name: yunion-ingress-controller-manager-config
        name: kubernetes-config
`
)
