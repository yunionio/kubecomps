REGISTRY ?= registry.cn-beijing.aliyuncs.com/yunionio
DOCKER_BUILD = docker build -t $(REGISTRY)
DOCKER_BUILDX = docker buildx build --platform linux/arm64,linux/amd64 --push -t $(REGISTRY)

helm:
	docker build -f ./Dockerfile.helm -t registry.cn-beijing.aliyuncs.com/yunionio/helm:v3.5.2 .

kube-build-img?=registry.cn-beijing.aliyuncs.com/yunionio/kube-build:3.16.0-go-1.18.2-0

kube-build:
	docker buildx build --platform linux/arm64,linux/amd64 -f ./Dockerfile.kube-build -t $(kube-build-img) . --push
	docker pull $(kube-build-img)

KUBECTL_VERSION = 1.24.3
kubectl:
	$(DOCKER_BUILDX)/kubectl:$(KUBECTL_VERSION) -f ./Dockerfile.kubectl-1_20 .

KUBESERVER_BASE_VERSION = 20230803.0
kubeserver-base:
	$(DOCKER_BUILDX)/kubeserver-base:$(KUBESERVER_BASE_VERSION) -f ./Dockerfile.kubeserver-base-1_20 .
