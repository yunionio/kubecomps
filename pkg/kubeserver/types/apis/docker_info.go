package apis

const (
	DockerInfoType                    = "dockerInfo"
	DockerInfoFieldArchitecture       = "architecture"
	DockerInfoFieldCgroupDriver       = "cgroupDriver"
	DockerInfoFieldDebug              = "debug"
	DockerInfoFieldDockerRootDir      = "dockerRootDir"
	DockerInfoFieldDriver             = "driver"
	DockerInfoFieldExperimentalBuild  = "experimentalBuild"
	DockerInfoFieldHTTPProxy          = "httpProxy"
	DockerInfoFieldHTTPSProxy         = "httpsProxy"
	DockerInfoFieldIndexServerAddress = "indexServerAddress"
	DockerInfoFieldKernelVersion      = "kernelVersion"
	DockerInfoFieldLabels             = "labels"
	DockerInfoFieldLoggingDriver      = "loggingDriver"
	DockerInfoFieldName               = "name"
	DockerInfoFieldNoProxy            = "noProxy"
	DockerInfoFieldOSType             = "osType"
	DockerInfoFieldOperatingSystem    = "operatingSystem"
	DockerInfoFieldServerVersion      = "serverVersion"
)

type DockerInfo struct {
	Architecture       string   `json:"architecture"`
	CgroupDriver       string   `json:"cgroupDriver"`
	Debug              bool     `json:"debug"`
	DockerRootDir      string   `json:"dockerRootDir"`
	Driver             string   `json:"driver"`
	ExperimentalBuild  bool     `json:"experimentalBuild"`
	HTTPProxy          string   `json:"httpProxy"`
	HTTPSProxy         string   `json:"httpsProxy"`
	IndexServerAddress string   `json:"indexServerAddress"`
	KernelVersion      string   `json:"kernelVersion"`
	Labels             []string `json:"labels"`
	LoggingDriver      string   `json:"loggingDriver"`
	Name               string   `json:"name"`
	NoProxy            string   `json:"noProxy"`
	OSType             string   `json:"osType"`
	OperatingSystem    string   `json:"operatingSystem"`
	ServerVersion      string   `json:"serverVersion"`
}
