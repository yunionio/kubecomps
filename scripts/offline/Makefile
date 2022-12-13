IMAGE_TAG=v0.0.2

registry-image:
	docker buildx build --platform=linux/amd64,linux/arm64 -t registry.cn-beijing.aliyuncs.com/zexi/kubeserver-offline-registry:$(IMAGE_TAG) -f ./registry/Dockerfile . --push

sync-files:
	./download.sh file

sync-images:
	./download.sh image

sync-files-images: sync-files sync-images

sync-rpms:
	bash -x ./rpm-maker.sh

nginx-image:
	docker buildx build --platform=linux/amd64,linux/arm64 -t registry.cn-beijing.aliyuncs.com/zexi/kubeserver-offline-nginx:$(IMAGE_TAG) -f ./nginx/Dockerfile . --push

image: registry-image nginx-image

.PHONY: registry-image nginx-image image
