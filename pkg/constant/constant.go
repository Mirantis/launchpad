package constant

const (
	// ImageRepo is the default image repo to use
	ImageRepo = "docker.io/mirantis"
	// ImageRepoLegacy is the default image repo to use for older versions
	ImageRepoLegacy = "docker.io/docker"
	// MKEVersion is the default MKE version to use
	MKEVersion = "3.3.3"
	// MSRVersion is the default MSR version to use
	MSRVersion = "2.8.3"
	// EngineVersion is the default engine version
	EngineVersion = "19.03.12"
	// EngineChannel is the default engine channel
	EngineChannel = "stable"
	// EngineRepoURL is the default engine repo
	EngineRepoURL = "https://repos.mirantis.com"
	// EngineInstallURLLinux is the default engine install script location for linux
	EngineInstallURLLinux = "https://get.mirantis.com/"
	// EngineInstallURLWindows is the default engine install script location for windows
	EngineInstallURLWindows = "https://get.mirantis.com/install.ps1"
	// StateBaseDir defines the base dir for all local state
	StateBaseDir = ".mirantis-launchpad"
	// ManagedLabelCmd marks the node as being managed by launchpad
	ManagedLabelCmd = "node update --label-add com.mirantis.launchpad.managed=true"
	// ManagedMSRLabelCmd marks a MSR node as being managed by launchpad
	ManagedMSRLabelCmd = "node update --label-add com.mirantis.launchpad.managed.dtr=true"
)
