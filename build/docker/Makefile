REGISTRY ?= registry.cn-beijing.aliyuncs.com/yunionio
DOCKER_BUILD = docker build -t $(REGISTRY)
DOCKER_BUILDX = docker buildx build --platform linux/arm64,linux/amd64 --push -t $(REGISTRY)

helm:
	docker build -f ./Dockerfile.helm -t registry.cn-beijing.aliyuncs.com/yunionio/helm:v3.5.2 .

kube-build-img?=registry.cn-beijing.aliyuncs.com/yunionio/kube-build:3.16.0-go-1.18.2-0

kube-build:
	docker buildx build --platform linux/arm64,linux/amd64 -f ./Dockerfile.kube-build -t $(kube-build-img) . --push
	docker pull $(kube-build-img)

KUBESERVER_BASE_VERSION = v0.0.1

kubeserver-base:
	$(DOCKER_BUILDX)/kubeserver-base:$(KUBESERVER_BASE_VERSION) -f ./Dockerfile.kubeserver-base .
