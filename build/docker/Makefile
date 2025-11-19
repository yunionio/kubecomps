REGISTRY ?= registry.cn-beijing.aliyuncs.com/yunionio
DOCKER_BUILD = docker build -t $(REGISTRY)
platforms = linux/arm64,linux/amd64,linux/riscv64
DOCKER_BUILDX = docker buildx build --platform $(platforms) --push -t $(REGISTRY)
KUBE_BUILD_VERSION = 3.22.2-go-1.24.9-0
HELM_VERSION = v4.0.0-1

helm:
	$(DOCKER_BUILDX)/helm:$(HELM_VERSION) -f ./Dockerfile.helm .
	docker pull $(REGISTRY)/helm:$(HELM_VERSION)

kube-build:
	$(DOCKER_BUILDX)/kube-build:$(KUBE_BUILD_VERSION) -f ./Dockerfile.kube-build .
	docker pull $(REGISTRY)/kube-build:$(KUBE_BUILD_VERSION)s

KUBECTL_VERSION = 1.24.3-1
kubectl:
	$(DOCKER_BUILDX)/kubectl:$(KUBECTL_VERSION) -f ./Dockerfile.kubectl-1_20 .
	docker pull $(REGISTRY)/kubectl:$(KUBECTL_VERSION)

KUBESERVER_BASE_VERSION = 20251118.0
kubeserver-base:
	$(DOCKER_BUILDX)/kubeserver-base:$(KUBESERVER_BASE_VERSION) -f ./Dockerfile.kubeserver-base-1_20 .
