package addons

const (
	HelmTemplate = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tiller
  namespace: kube-system

---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: tiller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: tiller
    namespace: kube-system

---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  labels:
    app: helm
    name: tiller
    lxcfs: "false"
  name: tiller-deploy
  namespace: kube-system
spec:
  template:
    metadata:
      labels:
        app: helm
        name: tiller
        lxcfs: "false"
    spec:
      automountServiceAccountToken: true
      containers:
      - env:
        - name: TILLER_NAMESPACE
          value: kube-system
        - name: TILLER_HISTORY_MAX
          value: "0"
        image: {{ .TillerImage }}
        livenessProbe:
          failureThreshold: 3
          httpGet:
            path: /liveness
            port: 44135
            scheme: HTTP
          initialDelaySeconds: 1
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 1
        name: tiller
        ports:
        - containerPort: 44134
          name: tiller
          protocol: TCP
        - containerPort: 44135
          name: http
          protocol: TCP
        readinessProbe:
          failureThreshold: 3
          httpGet:
            path: /readiness
            port: 44135
            scheme: HTTP
          initialDelaySeconds: 1
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 1
      serviceAccount: tiller
      serviceAccountName: tiller
`
)
