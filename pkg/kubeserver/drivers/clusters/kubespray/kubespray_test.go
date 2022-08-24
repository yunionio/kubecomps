package kubespray

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"yunion.io/x/kubecomps/pkg/kubeserver/constants"
)

func TestKubesprayInventoryHost(t *testing.T) {
	Convey("Test kubespray inventory host item", t, func() {

		Convey("Check role", func() {
			r, _ := NewKubesprayInventoryHost("node1", "192.168.1.1", "root", "", KubesprayNodeRoleMaster, KubesprayNodeRoleEtcd)
			So(r.HasRole(""), ShouldBeFalse)
			So(r.HasRole(KubesprayNodeRoleMaster), ShouldBeTrue)
			So(r.IsEtcdMember(), ShouldBeTrue)
			So(r.GetEtcdMemberName(), ShouldEqual, "node1")
		})

		Convey("Check ToString", func() {
			r, _ := NewKubesprayInventoryHost("node1", "192.168.1.1", "root", "123", KubesprayNodeRoleMaster, KubesprayNodeRoleEtcd)
			r.Ip = "10.168.0.2"
			str, err := r.ToString()
			So(err, ShouldBeNil)
			So(str, ShouldEqual, "node1\tansible_host=192.168.1.1\tansible_ssh_user=root\tansible_ssh_pass=123\tip=10.168.0.2\tetcd_member_name=node1")
		})
	})
}

func TestKubesprayInventory(t *testing.T) {
	Convey("Test inventory", t, func() {

		newH := func(hostname, host string, roles ...KubesprayNodeRole) *KubesprayInventoryHost {
			h, _ := NewKubesprayInventoryHost(hostname, host, "root", "passwd", roles...)
			return h
		}

		Convey("Test multi role", func() {
			h1 := newH("node1", "192.168.2.1", KubesprayNodeRoleMaster, KubesprayNodeRoleEtcd, KubesprayNodeRoleNode, KubesprayNodeRoleControlPlane)
			h2 := newH("node2", "192.168.2.2", KubesprayNodeRoleNode)
			iv := KubesprayInventory{
				kubeVersion: constants.K8S_VERSION_1_17_0,
				Hosts:       []*KubesprayInventoryHost{h1, h2},
			}

			ivExpStr := "[all]\n"
			ivExpStr += "node1\tansible_host=192.168.2.1\tansible_ssh_user=root\tansible_ssh_pass=passwd\tetcd_member_name=node1\n"
			ivExpStr += "node2\tansible_host=192.168.2.2\tansible_ssh_user=root\tansible_ssh_pass=passwd\n"
			ivExpStr += "\n[kube-master]\n"
			ivExpStr += "node1\n"
			ivExpStr += "\n[kube-control-plane]\n"
			ivExpStr += "node1\n"
			ivExpStr += "\n[etcd]\n"
			ivExpStr += "node1\n"
			ivExpStr += "\n[kube-node]\n"
			ivExpStr += "node1\nnode2\n"
			ivExpStr += "\n[calico-rr]\n"
			ivExpStr += "\n[k8s-cluster:children]\nkube-master\nkube-control-plane\nkube-node\ncalico-rr"

			str, err := iv.ToString()
			So(err, ShouldBeNil)
			So(str, ShouldEqual, ivExpStr)
		})
	})
}
