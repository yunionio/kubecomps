package plugin

import (
	"net"
	"reflect"
	"testing"

	"github.com/containernetworking/cni/pkg/types"
)

func Test_parseCNIArgs(t *testing.T) {
	tests := []struct {
		args    string
		want    *PodInfo
		wantErr bool
	}{
		{
			args: "IgnoreUnknown=1;K8S_POD_NAMESPACE=27c9464ab54947328a29298761895be3;K8S_POD_NAME=test-pod5;K8S_POD_INFRA_CONTAINER_ID=c73d1df43df96b6804330a257855a5c2c8355d3f84019c57bcc8b5ede14a11ed;K8S_POD_UID=e25e38ef-fe98-4993-8641-699cd0530fc0",
			want: &PodInfo{
				Namespace:   "27c9464ab54947328a29298761895be3",
				Name:        "test-pod5",
				ContainerId: "c73d1df43df96b6804330a257855a5c2c8355d3f84019c57bcc8b5ede14a11ed",
				Id:          "e25e38ef-fe98-4993-8641-699cd0530fc0",
			},
			wantErr: false,
		},
		{
			args:    "",
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.args, func(t *testing.T) {
			got, err := NewPodInfoFromCNIArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPodInfoFromCNIArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewPodInfoFromCNIArgs() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getNetworkConfig_metadataRoutes(t *testing.T) {
	mustCIDR := func(s string) net.IPNet {
		_, n, err := net.ParseCIDR(s)
		if err != nil {
			t.Fatalf("ParseCIDR(%q): %v", s, err)
		}
		return *n
	}

	nic := PodNic{
		Interface: "eth0",
		Mac:       "00:11:22:33:44:55",
		Ip:        "192.168.1.100",
		Masklen:   24,
		Gateway:   "192.168.1.1",
	}

	t.Run("defaultGw true ipv4 only", func(t *testing.T) {
		_, _, routes, err := getNetworkConfig(0, nic, true)
		if err != nil {
			t.Fatalf("getNetworkConfig: %v", err)
		}
		want := []*types.Route{
			{Dst: mustCIDR("0.0.0.0/0"), GW: net.ParseIP("192.168.1.1")},
			{Dst: mustCIDR("169.254.169.254/32"), GW: net.ParseIP("0.0.0.0")},
		}
		if !routesEqual(routes, want) {
			t.Errorf("routes = %#v, want %#v", routes, want)
		}
	})

	t.Run("defaultGw false no metadata route", func(t *testing.T) {
		_, _, routes, err := getNetworkConfig(1, nic, false)
		if err != nil {
			t.Fatalf("getNetworkConfig: %v", err)
		}
		if len(routes) != 0 {
			t.Errorf("routes = %#v, want empty", routes)
		}
	})

	t.Run("defaultGw true with ipv6", func(t *testing.T) {
		nic6 := nic
		nic6.Ip6 = "2001:db8::100"
		nic6.Masklen6 = 64
		nic6.Gateway6 = "2001:db8::1"

		_, _, routes, err := getNetworkConfig(0, nic6, true)
		if err != nil {
			t.Fatalf("getNetworkConfig: %v", err)
		}
		want := []*types.Route{
			{Dst: mustCIDR("0.0.0.0/0"), GW: net.ParseIP("192.168.1.1")},
			{Dst: mustCIDR("169.254.169.254/32"), GW: net.ParseIP("0.0.0.0")},
			{Dst: mustCIDR("::/0"), GW: net.ParseIP("2001:db8::1")},
			{Dst: mustCIDR("fd00:ec2::254/128"), GW: net.ParseIP("::")},
		}
		if !routesEqual(routes, want) {
			t.Errorf("routes = %#v, want %#v", routes, want)
		}
	})
}

func routesEqual(got, want []*types.Route) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i].Dst.String() != want[i].Dst.String() {
			return false
		}
		if !got[i].GW.Equal(want[i].GW) {
			return false
		}
	}
	return true
}
