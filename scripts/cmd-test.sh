#!/bin/bash

# climc --debug k8s-cluster-create --resource-type guest --provider onecloud --machine vm:controlplane --machine-hypervisor aliyun --image-repo registry.cn-beijing.aliyuncs.com/yunionio --vpc aliy --machine-sku 'ecs.c6.large' --machine-net lzx-subnet-2  ali-test

# climc --debug k8s-cluster-create --resource-type guest --provider onecloud \
    # --machine vm:controlplane \
    # --machine vm:node \
    # --machine-hypervisor kvm --image-repo registry.cn-beijing.aliyuncs.com/yunionio --vpc lzx-vpc --machine-net lzx-test-vpc-subnet1 --machine-cpu 4 --machine-disk CentOS-7.6.1810-20190430.qcow2:30g --machine-memory 4g kvm-test2

climc --debug k8s-cluster-create --resource-type guest --provider onecloud \
    --machine vm:controlplane \
    --machine vm:node \
    --machine-hypervisor kvm --image-repo registry.cn-beijing.aliyuncs.com/yunionio --vpc default --machine-net GUEST-NET190 --machine-cpu 4 --machine-disk CentOS-7.6.1810-20190430.qcow2:30g --machine-memory 4g kvm-test

