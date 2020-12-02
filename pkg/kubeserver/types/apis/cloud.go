package apis

type CloudHost struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	ManagerUrl string `json:"manager_uri"`
	HostType   string `json:"host_type"`
	HostStatus string `json:"host_status"`
	AccessIp   string `json:"access_ip"`
	Status     string `json:"status"`
	Enabled    bool   `json:"enabled"`
}
