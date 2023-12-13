package constant

const (
	// ImageRepo is the default image repo to use.
	ImageRepo = "docker.io/mirantis"
	// ImageRepoLegacy is the default image repo to use for older versions.
	ImageRepoLegacy = "docker.io/docker"
	// MCRVersion is the default engine version.
	MCRVersion = "20.10.13"
	// MCRChannel is the default engine channel.
	MCRChannel = "stable"
	// MCRRepoURL is the default engine repo.
	MCRRepoURL = "https://repos.mirantis.com"
	// MCRInstallURLLinux is the default engine install script location for linux.
	MCRInstallURLLinux = "https://get.mirantis.com/"
	// MCRInstallURLWindows is the default engine install script location for windows.
	MCRInstallURLWindows = "https://get.mirantis.com/install.ps1"
	// StateBaseDir defines the base dir for all local state.
	StateBaseDir = ".mirantis-launchpad"
	// ManagedLabelCmd marks the node as being managed by launchpad.
	ManagedLabelCmd = "node update --label-add com.mirantis.launchpad.managed=true"
	// ManagedMSRLabelCmd marks a MSR node as being managed by launchpad.
	ManagedMSRLabelCmd = "node update --label-add com.mirantis.launchpad.managed.dtr=true"
	// LinuxDefaultDockerRoot defines the default docker root.
	LinuxDefaultDockerRoot = "/var/lib/docker"
	// LinuxDefaultDockerExecRoot defines the default docker exec root.
	LinuxDefaultDockerExecRoot = "/var/run/docker"
	// LinuxDefaultDockerDaemonPath defines the default docker daemon path.
	LinuxDefaultDockerDaemonPath = "/etc/docker/daemon.json"
	// LinuxDefaultRootlessDockerDaemonPath defines the default rootless docker daemon path.
	LinuxDefaultRootlessDockerDaemonPath = "~/.config/docker/daemon.json"
	// WindowsDefaultDockerRoot defines the default windows docker root.
	WindowsDefaultDockerRoot = "C:\\ProgramData\\Docker"
	// MSROperator is the name of the MSR operator.
	MSROperator = "msr-operator"
	// PostgresOperator is the name of the postgres operator.
	PostgresOperator = "postgres-operator"
	// CertManager is the name of the cert manager.
	CertManager = "cert-manager"
	// RethinkDBOperator is the name of the rethinkdb operator.
	RethinkDBOperator = "rethinkdb-operator"
	// KubeConfigFile is the name of the kubeconfig file.
	KubeConfigFile = "kube.yml"
)
