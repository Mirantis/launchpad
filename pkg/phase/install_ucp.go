package phase

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/ucp"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta1"
	log "github.com/sirupsen/logrus"
)

const configName string = "com.docker.ucp.config"

var (
	ports = append([]int{
		179,
		443,
		2376,
		6443,
		6444,
		10250,
		12376,
		12388,
	}, intRange(12378, 12386)...)
)

// InstallUCP is the phase implementation for running the actual UCP installer container
type InstallUCP struct {
	Analytics
}

// Title prints the phase title
func (p *InstallUCP) Title() string {
	return "Install UCP components"
}

// Run the installer container
func (p *InstallUCP) Run(config *api.ClusterConfig) error {
	props := analytics.NewAnalyticsEventProperties()
	props["ucp_version"] = config.Spec.Ucp.Version
	p.EventProperties = props
	swarmLeader := config.Spec.SwarmLeader()

	if config.Spec.Ucp.Metadata.Installed {
		log.Infof("%s: UCP already installed at version %s, not running installer", swarmLeader.Address, config.Spec.Ucp.Metadata.InstalledVersion)
		return nil
	}

	image := fmt.Sprintf("%s/ucp:%s", config.Spec.Ucp.ImageRepo, config.Spec.Ucp.Version)
	installFlags := config.Spec.Ucp.InstallFlags
	if config.Spec.Ucp.ConfigData != "" {
		defer func() {
			err := swarmLeader.Execf("sudo docker config rm %s", configName)
			if err != nil {
				log.Warnf("Failed to remove the temporary UCP installer configuration %s : %s", configName, err)
			}
		}()

		installFlags = append(installFlags, "--existing-config")
		log.Info("Creating UCP configuration")
		configCmd := swarmLeader.Configurer.DockerCommandf("config create %s -", configName)
		err := swarmLeader.ExecCmd(configCmd, config.Spec.Ucp.ConfigData, false, false)
		if err != nil {
			return err
		}
	}

	if licenseFilePath := config.Spec.Ucp.LicenseFilePath; licenseFilePath != "" {
		log.Debugf("Installing with LicenseFilePath: %s", licenseFilePath)
		license, err := ioutil.ReadFile(licenseFilePath)
		if err != nil {
			return fmt.Errorf("error while reading license file %s: %v", licenseFilePath, err)
		}
		installFlags = append(installFlags, fmt.Sprintf("--license '%s'", string(license)))
	}

	if config.Spec.Ucp.IsCustomImageRepo() {
		// In case of custom repo, don't let UCP to check the images
		installFlags = append(installFlags, "--pull never")
	}
	runFlags := []string{"--rm", "-i", "-v /var/run/docker.sock:/var/run/docker.sock"}
	if swarmLeader.Configurer.SELinuxEnabled() {
		runFlags = append(runFlags, "--security-opt label=disable")
	}
	
	log.Debugln("Starting to check ports for connectivity")
	if err := assurePortsAreAccessible(swarmLeader, image); err != nil {
		return NewError(err.Error())
	}
	log.Debugln("All needed ports are accessible")

	installCmd := swarmLeader.Configurer.DockerCommandf("run %s %s install %s", strings.Join(runFlags, " "), image, strings.Join(installFlags, " "))
	err := swarmLeader.ExecCmd(installCmd, "", true, true)
	if err != nil {
		return NewError("Failed to run UCP installer")
	}

	ucpMeta, err := ucp.CollectUcpFacts(swarmLeader)
	if err != nil {
		return fmt.Errorf("%s: failed to collect existing UCP details: %s", swarmLeader.Address, err.Error())
	}
	config.Spec.Ucp.Metadata = ucpMeta

	return nil
}

func assurePortsAreAccessible(host *api.Host, image string) (err error) {
	defer host.Exec("sudo docker rm -f port-tester")

	flags := []string{}
	for _, port := range ports {
		flags = append(flags, fmt.Sprintf("-p %d:2376", port))
	}
	cmd := fmt.Sprintf("sudo docker run -d --name port-tester --rm %s %s port-check-server", strings.Join(flags, " "), image)

	if err := host.Exec(cmd); err != nil {
		return err
	}

	pErr := &portError{}
	for _, p := range ports {
		// only wait 1 second if there is no connectivity
		cmd := fmt.Sprintf(`sudo curl -s -o /dev/null -m 1 -w "%%{http_code}" http://%s:%d/`, host.Address, p)
		o, err := host.ExecWithOutput(cmd)
		if err != nil {
			err := fmt.Errorf("error while executing cmd `%s`: %w", cmd, err)
			pErr.append(p, err)
		} else if o != "200" {
			err := fmt.Errorf("error while curl-ling port %d: got response code %s", p, o)
			pErr.append(p, err)
		}
	}
	return pErr.errOrNil()
}

type portError struct {
	Errors []error
	Ports  []int
}

func (p *portError) Error() string {
	ports := strings.Trim(strings.Replace(fmt.Sprint(p.Ports), " ", ", ", -1), "[]")
	return fmt.Sprintf("the following ports need to be accessible before proceeding with the installation - %s", ports)
}

func (p *portError) append(port int, err error) {
	p.Ports = append(p.Ports, port)
	p.Errors = append(p.Errors, err)
}

func (p *portError) errOrNil() error {
	if len(p.Ports) == 0 && len(p.Errors) == 0 {
		return nil
	}
	return p
}

func intRange(begin, end int) []int {
	if end < begin {
		return []int{}
	}

	res := []int{}
	for i := begin; i <= end; i++ {
		res = append(res, i)
	}
	return res
}
