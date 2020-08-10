package constant

const (
	// ImageRepo is the default image repo to use
	ImageRepo = "docker.io/docker"
	// ImageRepoNew is the default image repo to use for newer versions
	ImageRepoNew = "docker.io/mirantis"
	// UCPVersion is the default UCP version to use
	UCPVersion = "3.3.1"
	// DTRVersion is the default DTR version to use
	DTRVersion = "2.8.1"
	// EngineVersion is the default engine version
	EngineVersion = "19.03.8"
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
