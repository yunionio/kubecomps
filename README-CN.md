# Kubecomps

[English](./README.md) | [简体中文](./README-CN.md)

Kubecomps 主要包含 [Cloudpods](https://github.com/yunionio/cloudpods) 管理 Kubernetes 相关的组件。

- [cmd/kubeserver](./cmd/kubeserver): 管理 Kubernetes 多集群的后端服务
- [cmd/calico-node-agent](./cmd/calico-node-agent): 基于 calico 的 VPC CNI node agent

## 编译

请先参考文档 [https://www.cloudpods.org/zh/docs/contribute/dev-env/](https://www.cloudpods.org/zh/docs/contribute/dev-env/) 搭建编译环境。

- 编译 kubeserver 二进制:

```bash
# 先自动生成代码
$ make generate

# 编译 kubeserver 二进制
$ make cmd/kubeserver
```

- 制作 kubeserver 镜像:

```bash
# - REGISTRY 是自己的镜像仓库
# - VERSION 对应镜像的 TAG
$ REGISTRY=registry.cn-beijing.aliyuncs.com/yunionio VERSION=dev-test make image kubeserver
```
