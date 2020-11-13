package serve

import (
	"reflect"
	"testing"

	"yunion.io/x/kubecomps/pkg/calico-node-agent/types"
)

func Test_getIPPoolName(t *testing.T) {
	type args struct {
		nodeName string
		pool     *types.NodeIPPool
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "node1-192-168-1-2-32",
			args: args{
				nodeName: "node1",
				pool:     &types.NodeIPPool{CIDR: "192.168.1.2/32"},
			},
			want: "node1-192-168-1-2-32",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getIPPoolName(tt.args.nodeName, tt.args.pool); got != tt.want {
				t.Errorf("getIPPoolName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getIPPoolNodeSelector(t *testing.T) {
	tests := []struct {
		name     string
		nodeName string
		want     string
	}{
		{
			name:     "selector with nodeName",
			nodeName: "hostname1",
			want:     "kubernetes.io/hostname == \"hostname1\"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getIPPoolNodeSelector(tt.nodeName); got != tt.want {
				t.Errorf("getIPPoolNodeSelector() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getIPPoolLabels(t *testing.T) {
	tests := []struct {
		name string
		want map[string]string
	}{
		{
			name: "pool label",
			want: map[string]string{
				"yunion.io/managed": "calico-node-agent",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getIPPoolLabels(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getIPPoolLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}
