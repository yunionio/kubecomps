package userdata

const (
	nodeCloudInit = `{{.Header}}
write_files:
-   path: /etc/docker/daemon.json
    owner: root:root
    permissions: '0644'
    encoding: "base64"
    content: |
      {{.DockerConfig | Base64Encode}}

-   path: /run/kubeadm-node.yaml
    owner: root:root
    permissions: '0640'
    content: |
      ---
{{.JoinConfiguration | Indent 6}}
`
)

// NodeInputCloudInit defines the context to generate a node user data
type NodeInputCloudInit struct {
	baseUserDataCloudInit

	DockerConfig      string
	JoinConfiguration string
}

// NewNodeCloudInit returns the user data string to be used on a node instance
func NewNodeCloudInit(input *NodeInputCloudInit) (string, error) {
	input.Header = cloudConfigHeader
	fMap := map[string]interface{}{
		"Base64Encode": templateBase64Encode,
		"Indent":       templateYAMLIndent,
	}
	return generateWithFuncs("node", nodeCloudInit, fMap, input)
}
