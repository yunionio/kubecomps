#!/bin/bash

set -o errexit
set -o pipefail

if [ "$DEBUG" == "true" ]; then
    set -ex ;export PS4='+(${BASH_SOURCE}:${LINENO}): ${FUNCNAME[0]:+${FUNCNAME[0]}(): }'
fi

readlink_mac() {
  cd `dirname $1`
  TARGET_FILE=`basename $1`

  # Iterate down a (possible) chain of symlinks
  while [ -L "$TARGET_FILE" ]
  do
    TARGET_FILE=`readlink $TARGET_FILE`
    cd `dirname $TARGET_FILE`
    TARGET_FILE=`basename $TARGET_FILE`
  done

  # Compute the canonicalized name by finding the physical path
  # for the directory we're in and appending the target file.
  PHYS_DIR=`pwd -P`
  REAL_PATH=$PHYS_DIR/$TARGET_FILE
}

get_current_arch() {
    local current_arch
    case $(uname -m) in
        x86_64)
            current_arch=amd64
            ;;
        aarch64)
            current_arch=arm64
            ;;
    esac
    echo $current_arch
}

pushd $(cd "$(dirname "$0")"; pwd) > /dev/null
readlink_mac $(basename "$0")
cd "$(dirname "$REAL_PATH")"
CUR_DIR=$(pwd)
SRC_DIR=$(cd .. && pwd)
popd > /dev/null

DOCKER_DIR="$SRC_DIR/build/docker"

# https://docs.docker.com/develop/develop-images/build_enhancements/
export DOCKER_BUILDKIT=1

REGISTRY=${REGISTRY:-docker.io/yunion}
TAG=${TAG:-latest}
CURRENT_ARCH=$(get_current_arch)
ARCH=${ARCH:-$CURRENT_ARCH}

build_bin() {
    local BUILD_ARCH=$2
    local BUILD_CGO=$3
    case "$1" in
        *)
            docker run --rm \
                -v $SRC_DIR:/root/go/src/yunion.io/x/kubecomps \
                -v $SRC_DIR/_output/alpine-build:/root/go/src/yunion.io/x/kubecomps/_output \
                -v $SRC_DIR/_output/alpine-build/_cache:/root/.cache \
                registry.cn-beijing.aliyuncs.com/yunionio/kube-build:1.0-5 \
                /bin/sh -c "set -ex; cd /root/go/src/yunion.io/x/kubecomps; $BUILD_ARCH $BUILD_CGO GOOS=linux make cmd/$1; chown -R $(id -u):$(id -g) _output"
            ;;
    esac
}


build_image() {
    local tag=$1
    local file=$2
    local path=$3
    docker build -t "$tag" -f "$2" "$3"
}

buildx_and_push() {
    local tag=$1
    local file=$2
    local path=$3
    local arch=$4
    docker buildx build -t "$tag" --platform "linux/$arch" -f "$2" "$3" --push
    docker pull "$tag"
}

push_image() {
    local tag=$1
    docker push "$tag"
}

get_image_name() {
    local component=$1
    local arch=$2
    local is_all_arch=$3
    local img_name="$REGISTRY/$component:$TAG"
    if [[ "$is_all_arch" == "true" || "$arch" == arm64 ]]; then
        img_name="${img_name}-$arch"
    fi
    echo $img_name
}

build_process() {
    local component=$1
    local image_name=$component
    local is_all_arch=$3
    local img_name=$(get_image_name $component $arch $is_all_arch)

    build_bin $component

    build_image $img_name $DOCKER_DIR/Dockerfile.$component $SRC_DIR
    push_image "$img_name"
}

build_process_with_buildx() {
    local component=$1
    local arch=$2
    local is_all_arch=$3
    local img_name=$(get_image_name $component $arch $is_all_arch)

    build_env="GOARCH=$arch "

    case "$component" in
        kubeserver)
            buildx_and_push $img_name $DOCKER_DIR/Dockerfile.$component-mularch $SRC_DIR $arch
            ;;
        *)
            build_bin $component $build_env
            buildx_and_push $img_name $DOCKER_DIR/Dockerfile.$component $SRC_DIR $arch
            ;;
    esac
}

general_build(){
    local component=$1
    # 如果未指定，则默认使用当前架构
    local arch=${2:-$current_arch}
    local is_all_arch=$3

    build_process_with_buildx $component $arch $is_all_arch
}

make_manifest_image() {
    local component=$1
    local img_name=$(get_image_name $component "" "false")
    docker manifest create --amend $img_name \
        $img_name-amd64 \
        $img_name-arm64
    docker manifest annotate $img_name $img_name-arm64 --arch arm64
    docker manifest push $img_name
}

ALL_COMPONENTS=$(ls cmd | grep -v '.*cli$' | xargs)

if [ "$#" -lt 1 ]; then
    echo "No component is specified~"
    echo "You can specify a component in [$ALL_COMPONENTS]"
    echo "If you want to build all components, specify the component to: all."
    exit
elif [ "$#" -eq 1 ] && [ "$1" == "all" ]; then
    echo "Build all onecloud-ee docker images"
    COMPONENTS=$ALL_COMPONENTS
else
    COMPONENTS=$@
fi

cd $SRC_DIR
mkdir -p $SRC_DIR/_output

for component in $COMPONENTS; do
    if [[ $component == *cli ]]; then
        echo "Please build image for climc"
        continue
    fi
    echo "Start to build component: $component"

    case "$ARCH" in
        all)
            for arch in "arm64" "amd64"; do
                general_build $component $arch "true"
            done
            make_manifest_image $component
            ;;
        *)
            general_build $component $ARCH "false"
            ;;
    esac
done
