/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package userdata

const (
	nodeBashScript = `{{.Header}}

{{.DockerScript}}

{{.OnecloudConfig}}

cat >/tmp/kubeadm-node.yaml <<EOF
---
apiVersion: kubeadm.k8s.io/v1beta1
kind: JoinConfiguration
discovery:
  bootstrapToken:
    token: "{{.BootstrapToken}}"
    apiServerEndpoint: "{{.ELBAddress}}:6443"
    caCertHashes:
      - "{{.CACertHash}}"
nodeRegistration:
  name: "$(hostname)"
  kubeletExtraArgs:
    cloud-provider: external
    read-only-port: "10255"
    pod-infra-container-image: registry.cn-beijing.aliyuncs.com/yunionio/pause-amd64:3.1
    feature-gates: "CSIPersistentVolume=true,MountPropagation=true,KubeletPluginsWatcher=true,VolumeScheduling=true"
    eviction-hard: "memory.available<100Mi,nodefs.available<2Gi,nodefs.inodesFree<5%"
EOF

kubeadm join --config /tmp/kubeadm-node.yaml
systemctl enable kubelet
`
)

// NodeInput defines the context to generate a node user data.
type NodeInput struct {
	*baseUserData

	BaseConfigure

	CACertHash     string
	BootstrapToken string
	ELBAddress     string
}

// NewNode returns the user data string to be used on a node instance.
func NewNode(input *NodeInput) (string, error) {
	var err error
	input.baseUserData, err = newBaseUserData(input.BaseConfigure)
	if err != nil {
		return "", err
	}
	return generate("node", nodeBashScript, input)
}
