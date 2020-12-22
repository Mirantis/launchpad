package constant

const (
	// ImageRepo is the default image repo to use
	ImageRepo = "docker.io/mirantis"
	// ImageRepoLegacy is the default image repo to use for older versions
	ImageRepoLegacy = "docker.io/docker"
	// MKEVersion is the default MKE version to use
	MKEVersion = "3.3.4"
	// MSRVersion is the default MSR version to use
	MSRVersion = "2.8.4"
	// MCRVersion is the default engine version
	MCRVersion = "19.03.14"
	// MCRChannel is the default engine channel
	MCRChannel = "stable"
	// MCRRepoURL is the default engine repo
	MCRRepoURL = "https://repos.mirantis.com"
	// MCRInstallURLLinux is the default engine install script location for linux
	MCRInstallURLLinux = "https://get.mirantis.com/"
	// MCRInstallURLWindows is the default engine install script location for windows
	MCRInstallURLWindows = "https://get.mirantis.com/install.ps1"
	// StateBaseDir defines the base dir for all local state
	StateBaseDir = ".mirantis-launchpad"
	// ManagedLabelCmd marks the node as being managed by launchpad
	ManagedLabelCmd = "node update --label-add com.mirantis.launchpad.managed=true"
	// ManagedMSRLabelCmd marks a MSR node as being managed by launchpad
	ManagedMSRLabelCmd = "node update --label-add com.mirantis.launchpad.managed.dtr=true"
)
