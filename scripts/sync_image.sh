#!/bin/bash

set -ex

DEST_REGISTRY=${DEST_REGISTRY:-registry.cn-beijing.aliyuncs.com/yunionio}

sync_img() {
    local source_registry=$1
    local name=$2
    local tag=$3
    local platform=$4

    local img=$source_registry/$name:$tag
    local dest_img=$DEST_REGISTRY/$name:$tag
    local dest_img_platform=$dest_img-$platform

    docker pull $img --platform $platform
    docker tag $img $dest_img_platform
    docker push $dest_img_platform
}

sync_x86_arm_64_img() {
    local source_registry=$1
    local name=$2
    local tag=$3

    local dest_img=$DEST_REGISTRY/$name:$tag

    sync_img $source_registry $name $tag amd64
    sync_img $source_registry $name $tag arm64

    docker manifest create $dest_img $dest_img-amd64 $dest_img-arm64
    docker manifest annotate $dest_img $dest_img-amd64 --arch amd64
    docker manifest annotate $dest_img $dest_img-arm64 --arch arm64

    docker manifest push $dest_img
}

# sync_x86_arm_64_img grafana grafana 6.7.1
# sync_x86_arm_64_img kiwigrid k8s-sidecar 1.12.2
# sync_x86_arm_64_img grafana loki 2.2.1
# sync_x86_arm_64_img grafana promtail 2.2.1
# sync_x86_arm_64_img prom prometheus v2.28.1
# sync_x86_arm_64_img carlosedp prometheus-operator v0.37.0
# sync_x86_arm_64_img liaoronghui prometheus-config-reloader v0.38.1
# sync_x86_arm_64_img prom node-exporter v1.2.0
# sync_x86_arm_64_img prom alertmanager v0.22.2
# sync_x86_arm_64_img k8s.gcr.io/kube-state-metrics kube-state-metrics v1.9.8
# sync_x86_arm_64_img minio minio RELEASE.2021-06-17T00-10-46Z
# sync_x86_arm_64_img minio mc RELEASE.2021-06-13T17-48-22Z
# sync_x86_arm_64_img raspbernetes thanos v0.22.0
# sync_x86_arm_64_img jimmidyson configmap-reload v0.5.0
# sync_x86_arm_64_img jettech kube-webhook-certgen v1.5.2
# sync_x86_arm_64_img ghostunnel ghostunnel v1.5.3

# ceph related
sync_x86_arm_64_img quay.io/ceph ceph v14.2.22
