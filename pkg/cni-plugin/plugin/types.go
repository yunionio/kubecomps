package plugin

import "fmt"

type PodDesc struct {
	Id     string   `json:"id"`
	Name   string   `json:"name"`
	Status string   `json:"status"`
	Nics   []PodNic `json:"nics"`
}

const (
	POD_NIC_PROVIDER_OVN = "ovn"
)

type PodNicVpc struct {
	Id           string `json:"id"`
	MappedIpAddr string `json:"mapped_ip_addr"`
	Provider     string `json:"provider"`
}

type PodNic struct {
	Index     int        `json:"index"`
	Bridge    string     `json:"bridge"`
	Ifname    string     `json:"ifname"`
	Interface string     `json:"interface"`
	Ip        string     `json:"ip"`
	Mac       string     `json:"mac"`
	Gateway   string     `json:"gateway"`
	Bandwidth int        `json:"bw"`
	Dns       string     `json:"dns"`
	Mtu       int        `json:"mtu"`
	Masklen   int        `json:"masklen,omitempty"`
	Domain    string     `json:"domain,omitempty"`
	NetId     string     `json:"net_id"`
	WireId    string     `json:"wire_id"`
	Vpc       *PodNicVpc `json:"vpc,omitempty"`
}

func (n PodNic) GetInterface(idx int) string {
	defaultName := fmt.Sprintf("eth%d", idx)
	return defaultName
}
