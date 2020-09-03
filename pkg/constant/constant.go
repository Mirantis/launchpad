package constant

const (
	// ImageRepo is the default image repo to use
	ImageRepo = "docker.io/mirantis"
	// ImageRepoLegacy is the default image repo to use for older versions
	ImageRepoLegacy = "docker.io/docker"
	// UCPVersion is the default UCP version to use
	UCPVersion = "3.3.2"
	// DTRVersion is the default DTR version to use
	DTRVersion = "2.8.2"
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
	// ProductDTR defines DTR
	ProductDTR = "dtr"
	// ManagedLabelCmd marks the node as being managed by launchpad
	ManagedLabelCmd = "node update --label-add com.mirantis.launchpad.managed=true"
	// ManagedDtrLabelCmd marks a DTR node as being managed by launchpad
	ManagedDtrLabelCmd = "node update --label-add com.mirantis.launchpad.managed.dtr=true"
)
