package constant

const (
	InternalCdnBaseUrl = "https://artifactory.mcp.mirantis.net/binary-dev-kaas-virtual"
	InternalEuCdnBaseUrl = "https://artifactory.mirantis.com/binary-kaas-local"
	PublicCICdnBaseUrl = "https://binary-dev-kaas-mirantis-com.s3.amazonaws.com"
	PublicCdnBaseUrl = "https://binary.mirantis.com"
	DefaultReleasesPath = "releases"
	LatestKaaSRelease = "1.12.0"
	BootstrapEnvFile = "bootstrap.env"
	DefaultKaaSReleasesPath = "kaas"
	DefaultClusterReleasesPath = "cluster"
	DefaultCDNRegion = "public"
	DefaultTargetDir = "kaas-bootstrap"
	KaaSReleasesPath = "releases/kaas"
	ClusterReleasesPath = "releases/cluster"
)

var LatestClusterReleases = []string{"3.8.0", "4.5.0", "4.6.0", "5.5.0", "5.6.0"}

