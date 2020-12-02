package kubeadm

func GetControlplaneInitScript() string {
	return `
#!/usr/bin/env bash

sudo systemctl enable docker kubelet
sudo systemctl restart docker
sudo kubeadm init --config /run/kubeadm.yaml
`
}

func GetControlplaneJoinScript() string {
	return `
#!/usr/bin/env bash

sudo systemctl enable kubelet
sudo systemctl enable docker
sudo systemctl restart docker
sudo kubeadm join --config /run/kubeadm-controlplane-join-config.yaml
`
}

func GetNodeJoinScript() string {
	return `
#!/usr/bin/env bash

sudo systemctl enable kubelet
sudo systemctl enable docker
sudo systemctl restart docker
sudo kubeadm join --config /run/kubeadm-node.yaml
`
}
