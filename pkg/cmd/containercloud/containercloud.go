package containercloud

import (
	"fmt"
	"os"
	"path"
	"runtime"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/util"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type DownloadBootstrapBundle struct {
	TargetDir          string
	Region             string
	BaseURL            string
	ReleaseFile        string
	ClusterReleasesDir string
	ReleasesBaseURL    string
	BootstrapVersion   string
	BootstrapURL       string
	BootstrapTarball   string
}

// Initialize the downloader by assigning values based on the command flags and
// creating necessary directories and paths.
func (d *DownloadBootstrapBundle) Init() error {
	log.Infof("Initialized downloader, creating target directory.\n")
	if err := d.ensureTargetDir(); err != nil {
		return err
	}
	log.Debugf("Created target directory: %s", d.TargetDir)
	return nil
}

func (d *DownloadBootstrapBundle) Run() error {
	var err error
	if err = d.ensureLatestRelease(); err != nil {
		return err
	}
	d.BootstrapVersion, err = d.getBootstrapVersion()
	if err != nil {
		return err
	}
	d.BootstrapURL, err = d.getBootstrapURL()
	if err != nil {
		return err
	}
	if err = d.downloadBootstrapTarball(); err != nil {
		return err
	}
	if err = d.extractBootstrapTarball(); err != nil {
		return err
	}
	if err = d.writeBootstrapEnv(); err != nil {
		return err
	}
	return nil
}

// Make sure that the target dir is created, and contains all necessary
// sub-directories for release files.
func (d *DownloadBootstrapBundle) ensureTargetDir() error {
	if err := util.EnsureDir(d.TargetDir); err != nil {
		return err
	}
	kaasDir := path.Join(d.TargetDir, constant.KaaSReleasesPath)
	if err := util.EnsureDir(kaasDir); err != nil {
		return err
	}
	clusterDir := path.Join(d.TargetDir, constant.ClusterReleasesPath)
	if err := util.EnsureDir(clusterDir); err != nil {
		return err
	}
	return nil
}

// Download the latest release files to the target dir under the following structure:
// $TARGET_DIR/
// - releases/
//   - cluster/
//   - kaas/
func (d *DownloadBootstrapBundle) ensureLatestRelease() error {
	log.Infof("Preparing to download the latest release.\n")
	kaasReleaseFile := fmt.Sprintf("%s.yaml", constant.LatestKaaSRelease)
	kaasReleasePath := path.Join(constant.KaaSReleasesPath, kaasReleaseFile)
	kaasReleaseUrl := fmt.Sprintf("%s/%s", d.BaseURL, kaasReleasePath)
	dir := path.Join(d.TargetDir, constant.KaaSReleasesPath)
	log.Debugf("Using directory \"%s\" to download the release.\n", dir)
	log.Infof("Downloading release \"%s\"...\n", constant.LatestKaaSRelease)
	if err := util.DownloadFile(kaasReleaseUrl, dir); err != nil {
		return err
	}
	log.Infof("Download finished.\n")
	dir = path.Join(d.TargetDir, constant.ClusterReleasesPath)
	log.Debugf("Using directory \"%s\" to download the cluster releases.\n", dir)
	for _, r := range constant.LatestClusterReleases {
		clusterReleaseFile := fmt.Sprintf("%s.yaml", r)
		clusterReleasePath := path.Join(constant.ClusterReleasesPath, clusterReleaseFile)
		clusterReleaseUrl := fmt.Sprintf("%s/%s", d.BaseURL, clusterReleasePath)
		log.Infof("Downloading cluster release \"%s\"...", r)
		if err := util.DownloadFile(clusterReleaseUrl, dir); err != nil {
			return err
		}
		log.Infof("Download finished.\n")
	}
	return nil
}

// Get the bootstrap version from the KaaSRelease file. Tries to get the file
// path from the flags and environment. If not specified, attempt to download
// all release files for the latest version of DE Container Cloud from the known
// location.
func (d *DownloadBootstrapBundle) getBootstrapVersion() (string, error) {
	// Use the constant to get the file name for the lastest KaaS release
	// if the release file not specified through CLI flag of env var.
	if d.ReleaseFile == "" {
		f := fmt.Sprintf("%s.yaml", constant.LatestKaaSRelease)
		d.ReleaseFile = path.Join(d.TargetDir, constant.KaaSReleasesPath, f)
	}
	log.Debugf("Using release file \"%s\" to find bootstrap version.\n", d.ReleaseFile)
	version, err := readBootstrapVersionFromFile(d.ReleaseFile)
	if err != nil {
		return "", err
	}
	log.Infof("Found bootstrap version \"%s\".\n", version)
	return version, nil
}

// This function returns the URL that allows to download the bootstrap
// tarball. It gets all necessary parameters automatically from cli flags,
// env vars or defaults.
func (d *DownloadBootstrapBundle) getBootstrapURL() (string, error) {
	osTag, err := getOSTag()
	if err != nil {
		return "", err
	}
	if err != nil {
		return "", err
	}
	d.BootstrapTarball = fmt.Sprintf("bootstrap-%s-%s.tar.gz", osTag, d.BootstrapVersion)
	url := fmt.Sprintf("%s/core/bin/%s", d.BaseURL, d.BootstrapTarball)
	return url, nil
}

// Download the bootstrap tarball.
// TODO(ogelbukh): verify the tarball (using md5sum, other means)
func (d *DownloadBootstrapBundle) downloadBootstrapTarball() error {
	log.Debugf("Bootstrap bundle URL: %s\n", d.BootstrapURL)
	log.Infof("Downloading bootstrap bundle...\n")
	if err := util.DownloadFile(d.BootstrapURL, d.TargetDir); err != nil {
		return err
	}
	log.Infof("Bootstrap bundle download finished.\n")
	return nil
}

// Extract the bootstrap tarball.
func (d *DownloadBootstrapBundle) extractBootstrapTarball() error {
	p := path.Join(d.TargetDir, d.BootstrapTarball)
	log.Infof("Extracting bootstrap bundle to \"%s\" dir.\n", d.TargetDir)
	if err := util.ExtractTarball(p, d.TargetDir); err != nil {
		return err
	}
	log.Infof("Bootstrap bundle extracted.\n")
	return nil
}

// Write env variable to the bootstrap.env file.
func (d *DownloadBootstrapBundle) writeBootstrapEnv() error {
	fpath := path.Join(d.TargetDir, constant.BootstrapEnvFile)
	data := fmt.Sprintf("%s=%s\n", constant.KaaSReleasesYamlEnvVar, d.ReleaseFile)
	data += fmt.Sprintf("%s=%s\n", constant.ClusterReleasesDirEnvVar, d.ClusterReleasesDir)
	data += fmt.Sprintf("%s=%s\n", constant.KaaSCDNRegionEnvVar, d.Region)
	rawData := []byte(data)
	mode := os.FileMode(uint32(0644))
	if err := util.WriteFile(fpath, rawData, mode); err != nil {
		return err
	}
	return nil
}

// Read the bootstrap version from a YaML file that contains the KaaSRelease object
// and return.
func readBootstrapVersionFromFile(f string) (string, error) {
	data, err := util.LoadExternalFile(f)
	if err != nil {
		return "", err
	}
	cluster := make(map[interface{}]interface{})
	err = yaml.Unmarshal(data, &cluster)
	if err != nil {
		return "", err
	}
	spec := cluster["spec"]
	v := spec.(map[interface{}]interface{})["bootstrap"].(map[interface{}]interface{})["version"]
	if version, ok := v.(string); ok {
		return version, nil
	} else {
		return "", fmt.Errorf("Not a string in bootstrap version field: %v\n", version)
	}
}

// Get the operating system tag to use in the bootstrap URL.
func getOSTag() (string, error) {
	tag := runtime.GOOS
	switch tag {
	case "darwin":
		return tag, nil
	case "linux":
		return tag, nil
	default:
		err := fmt.Errorf("Unexpected operating system \"%s\", supported systems are \"darwin\", \"linux\".\n", tag)
		return tag, err
	}
}
