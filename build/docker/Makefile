helm:
	docker build -f ./Dockerfile.helm -t registry.cn-beijing.aliyuncs.com/yunionio/helm:v3.5.2 .

kube-build-img?=registry.cn-beijing.aliyuncs.com/yunionio/kube-build:1.0-5

kube-build:
	docker buildx build --platform linux/arm64,linux/amd64 -f ./Dockerfile.kube-build -t $(kube-build-img) . --push
	docker pull $(kube-build-img)
