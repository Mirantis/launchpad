package enterpriselinux

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/k0sproject/rig/exec"
	"github.com/k0sproject/rig/os"
	"github.com/k0sproject/rig/os/linux"

	"github.com/Mirantis/launchpad/pkg/configurer"
	common "github.com/Mirantis/launchpad/pkg/product/common/api"
)

// Configurer is the EL family specific implementation of a host configurer.
type Configurer struct {
	linux.EnterpriseLinux
	configurer.LinuxConfigurer
}

// InstallMKEBasePackages install all the needed base packages on the host.
func (c Configurer) InstallMKEBasePackages(h os.Host) error {
	if err := c.InstallPackage(h, "curl", "socat", "iptables", "iputils", "gzip", "openssh"); err != nil {
		return fmt.Errorf("failed to install base packages: %w", err)
	}
	return nil
}

// InstallMCR install Docker EE engine on Linux.
func (c Configurer) InstallMCR(h os.Host, scriptPath string, engineConfig common.MCRConfig) error {
	ver, verErr := configurer.ResolveLinux(h)
	if verErr != nil {
		return fmt.Errorf("could not discover Linux version information")
	}

	if isEC2 := c.isAWSInstance(h); !isEC2 {
		log.Debugf("%s: confirmed that this is not an AWS instance", h)
	} else if c.InstallPackage(h, "rh-amazon-rhui-client") == nil {
		log.Infof("%s: appears to be an AWS EC2 instance, installed rh-amazon-rhui-client", h)
	}

	// e.g. https://repos.mirantis.com/rhel/$releasever/$basearch/<update-channel>
	baseUrl := fmt.Sprintf("%s/%s/%s/%s/%s", engineConfig.RepoURL, ver.ID, "$releasever", "$basearch", engineConfig.Channel)
	// e.g. https://repos.mirantis.com/oraclelinux/gpg
	gpgUrl := fmt.Sprintf("%s/%s/gpg", engineConfig.RepoURL, ver.ID)
	elRepoFilePath := "/etc/yum.repos.d/docker-ee.repo"
	elRepoTemplate := `[mirantis]
name=Mirantis Container Runtime
baseurl=%s
enabled=1
gpgcheck=1
gpgkey=%s
`
	elRepo := fmt.Sprintf(elRepoTemplate, baseUrl, gpgUrl)

	if err := c.WriteFile(h, elRepoFilePath, elRepo, "0600"); err != nil {
		return fmt.Errorf("Could not write Yum repo file for MCR")
	}

	if err := c.InstallPackage(h, "docker.ee"); err != nil {
		return fmt.Errorf("Package manager could not install docker-ee")
	}
	return nil
}

// UninstallMCR uninstalls docker-ee engine.
func (c Configurer) UninstallMCR(h os.Host, _ string, engineConfig common.MCRConfig) error {
	info, getDockerError := c.GetDockerInfo(h)
	if engineConfig.Prune {
		defer c.CleanupLingeringMCR(h, info)
	}
	if getDockerError == nil {
		if err := h.Exec("docker system prune -f"); err != nil {
			return fmt.Errorf("prune docker: %w", err)
		}

		if err := c.StopService(h, "docker"); err != nil {
			return fmt.Errorf("stop docker: %w", err)
		}

		if err := c.StopService(h, "containerd"); err != nil {
			return fmt.Errorf("stop containerd: %w", err)
		}

		if err := h.Exec("yum remove -y docker-ee docker-ee-cli", exec.Sudo(h)); err != nil {
			return fmt.Errorf("remove docker-ee yum package: %w", err)
		}
	}

	return nil
}

// function to check if the host is an AWS instance - https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-identity-documents.html
func (c Configurer) isAWSInstance(h os.Host) bool {
	found, err := h.ExecOutput("curl -s -m 5 http://169.254.169.254/latest/dynamic/instance-identity/document | grep region")
	if err != nil {
		log.Debugf("%s: curl on local-linked AWS id document failed: %v", h, err)
		return false
	}
	return strings.Contains(found, "region")
}
