package constant

const (
	// ImageRepo is the default image repo to use.
	ImageRepo = "docker.io/mirantis"
	// ImageRepoLegacy is the default image repo to use for older versions.
	ImageRepoLegacy = "docker.io/docker"
	// MCRVersion is the default engine version.
	MCRVersion = "25.0"
	// MCRChannel is the default engine channel.
	MCRChannel = "stable"
	// MCRRepoURL is the default engine repo.
	MCRRepoURL = "https://repos.mirantis.com"
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
	// KubernetesOrchestratorTaint taints the node as NoExecute when using swarm.
	KubernetesOrchestratorTaint = "com.docker.ucp.orchestrator.kubernetes"
	// MSROperatorDeploymentLabels are the labels the msr-operator deployment uses.
	MSROperatorDeploymentLabels = "app.kubernetes.io/name=msr-operator"
	// KubeConfigFile is the name of the kubeconfig file.
	KubeConfigFile = "kube.yml"
	// MSRNodeSelector is the node selector for MSR nodes.
	MSRNodeSelector = "node-role.kubernetes.io/msr"
	// DefaultStorageClassAnnotation is the annotation to set a StorageClass to the default.
	DefaultStorageClassAnnotation = "storageclass.kubernetes.io/is-default-class"
)

const (
	// MSROperator is the name of the MSR operator.
	MSROperator = "msr-operator"
	// PostgresOperator is the name of the postgres operator.
	PostgresOperator = "postgres-operator"
	// CertManager is the name of the cert manager.
	CertManager = "cert-manager"
	// RethinkDBOperator is the name of the rethinkdb operator.
	RethinkDBOperator = "rethinkdb-operator"
)
