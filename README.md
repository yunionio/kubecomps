# Kubecomps

[English](./README.md) | [简体中文](./README-CN.md)

Kubecomps contains the components that use to manage Kubernetes of [Cloudpods](https://github.com/yunionio/cloudpods) .

- [cmd/kubeserver](./cmd/kubeserver): the backend server to manage multiple Kubernetes cluster
- [cmd/calico-node-agent](./cmd/calico-node-agent): the VPC CNI node agent based on calico

## Build

Please follow the [https://www.cloudpods.org/zh/docs/contribute/dev-env/](https://www.cloudpods.org/zh/docs/contribute/dev-env/) to set up the development environment first.

- Build kubeserver binary:

```bash
# generate embedded code
$ make generate

# build kubeserver binary
$ make cmd/kubeserver
```

- Build kubeserver image:

```bash
$ REGISTRY=registry.cn-beijing.aliyuncs.com/yunionio VERSION=dev-test make image kubeserver
```
