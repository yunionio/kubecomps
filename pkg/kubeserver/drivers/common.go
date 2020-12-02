package drivers

import (
	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

func GetControlplaneMachineDatas(clusterId string, data []*api.CreateMachineData) ([]*api.CreateMachineData, []*api.CreateMachineData) {
	controls := make([]*api.CreateMachineData, 0)
	nodes := make([]*api.CreateMachineData, 0)
	for _, d := range data {
		if len(clusterId) != 0 {
			d.ClusterId = clusterId
		}
		if d.Role == api.RoleTypeControlplane {
			controls = append(controls, d)
		} else {
			nodes = append(nodes, d)
		}
	}
	return controls, nodes
}
