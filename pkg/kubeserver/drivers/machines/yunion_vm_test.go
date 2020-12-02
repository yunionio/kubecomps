package machines

import (
	"testing"

	ocapi "yunion.io/x/onecloud/pkg/apis/compute"
)

func TestSYunionVMDriver_getCalicoNodeAgentDeployScript(t *testing.T) {
	tests := []struct {
		name    string
		addrs   []*ocapi.NetworkAddressDetails
		want    string
		wantErr bool
	}{
		{
			name: "should equal",
			addrs: []*ocapi.NetworkAddressDetails{
				{
					IpAddr: "192.168.0.2",
				},
				{
					IpAddr: "192.168.0.3",
				},
			},
			want: `mkdir -p /var/run/calico/
cat >/var/run/calico/node-agent-config.yaml<<EOF
ipPools:
- cidr: 192.168.0.2/32
- cidr: 192.168.0.3/32
proxyARPInterface: all

EOF`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &SYunionVMDriver{}
			got, err := d.getCalicoNodeAgentDeployScript(tt.addrs)
			if (err != nil) != tt.wantErr {
				t.Errorf("SYunionVMDriver.getCalicoNodeAgentDeployScript() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SYunionVMDriver.getCalicoNodeAgentDeployScript() = %v, want %v", got, tt.want)
			}
		})
	}
}
