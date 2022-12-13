#!/bin/bash

set -eo pipefail

CUR_DIR="$(pwd)"
OUTPUT_DIR="$CUR_DIR/_output"
REGISTRY_DIR="$CUR_DIR/registry"


download_files() {
    #wget -c -x -P _output/files -i files.list
    grep -v '^#' files.list | wget -c -x -P _output/files -i -
}

start_registry() {
    local name=registry
    docker rm -f $name
    docker run \
        -v "$OUTPUT_DIR/registry:/var/lib/registry" \
        -v "$REGISTRY_DIR/config.yml:/etc/docker/registry/config.yml" \
        --name $name --rm \
        -d \
        -p 15000:5000 \
        registry:2.8.1
}

sync_images() {
    local registry="127.0.0.1:15000"
    for image in $(grep -v '^#' images.list); do
        local target_loc=$registry/${image#*/}
        echo "copy $image => $target_loc"
        skopeo copy \
            --insecure-policy \
            --src-tls-verify=false \
            --dest-tls-verify=false \
            --override-os linux  \
            --multi-arch all \
            docker://${image} docker://$target_loc
            #docker://m.daocloud.io/${image} docker://$registry/${image#*/}
    done
}

# mirror_docker_ce() {
#     wget --mirror --no-parent \
#         https://download.docker.com/linux/centos/7/x86_64/stable/ # need last slash for --no-parent
#
#     curl -SL https://download.docker.com/linux/centos/gpg >download.docker.com/linux/centos/gpg || exit 1
#
#     rm -rf $OUTPUT_DIR/rpms/docker-ce
#     mkdir -p $OUTPUT_DIR/rpms/
#     mv download.docker.com/linux/centos $OUTPUT_DIR/rpms/docker-ce
#     rm -rf download.docker.com
# }

file_step() {
    download_files
}

image_step() {
    start_registry
    sync_images
}

all_steps() {
    file_step
    image_step
}

if [[ "$1" == "file" ]]; then
    file_step
elif [[ "$1" == "image" ]]; then
    image_step
else
    all_steps
fi
        
