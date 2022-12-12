#!/bin/bash

set -eo pipefail

OUTPUT_DIR=/output/rpms

yum-config-manager --add-repo https://mirrors.aliyun.com/docker-ce/linux/centos/docker-ce.repo
yum install -y createrepo
#repoquery -R --resolve --recursive docker-ce-19.03.15
yumdownloader --downloadonly \
    --downloaddir=$OUTPUT_DIR \
    --destdir=$OUTPUT_DIR \
    --resolve \
    docker-ce-19.03.15 docker-ce-cli-19.03.15 containerd.io-1.4.9

createrepo $OUTPUT_DIR
createrepo --update $OUTPUT_DIR
