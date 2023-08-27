#!/bin/bash

set -ex

# aws ecr get-login-password --region us-west-2 | skopeo login --username AWS --password-stdin 602401143452.dkr.ecr.us-west-2.amazonaws.com

# skopeo copy --override-os linux  --multi-arch all docker://602401143452.dkr.ecr.us-west-2.amazonaws.com/amazon-k8s-cni-init:v1.13.4 docker://registry.cn-beijing.aliyuncs.com/yunionio/amazon-k8s-cni-init:v1.13.4
#skopeo copy --override-os linux  --multi-arch all docker://602401143452.dkr.ecr.us-west-2.amazonaws.com/amazon-k8s-cni:v1.13.4 docker://registry.cn-beijing.aliyuncs.com/yunionio/amazon-k8s-cni:v1.13.4
# skopeo copy --override-os linux  --multi-arch all docker://registry.k8s.io/provider-aws/cloud-controller-manager:v1.23.0-alpha.0 docker://registry.cn-beijing.aliyuncs.com/yunionio/aws-cloud-controller-manager:v1.23.0-alpha.0
# skopeo copy --override-os linux  --multi-arch all docker://registry.cn-beijing.aliyuncs.com/yunionio/amazon-k8s-cni:v1.13.3 docker://registry.us-west-1.aliyuncs.com/yunion-dev/amazon-k8s-cni:v1.13.3
skopeo copy --override-os linux  --multi-arch all docker://registry.cn-beijing.aliyuncs.com/yunionio/amazon-k8s-cni-init:v1.13.3 docker://registry.us-west-1.aliyuncs.com/yunion-dev/amazon-k8s-cni-init:v1.13.3
