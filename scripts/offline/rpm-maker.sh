#!/bin/bash

set -eo pipefail

mirror_docker() {
    local arch=$1
    #local image=centos:7
    local image=registry.cn-beijing.aliyuncs.com/yunionio/centos-build:go-1.18.3-0

    docker run \
        --network host \
        --platform linux/$arch \
        -it --rm --name offline-centos7-$arch \
        -v $(pwd)/mirror-docker.sh:/mirror-docker.sh \
        -v $(pwd)/_output:/output \
        $image \
        /mirror-docker.sh
}

mirror_docker amd64
mirror_docker arm64
