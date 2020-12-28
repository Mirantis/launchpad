package ubuntu

import (
	"fmt"
	"os"
	"time"

	configurer "github.com/Mirantis/mcc/pkg/configurer/k0s"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/Mirantis/mcc/pkg/product/k0s/api"
	k0sutil "github.com/Mirantis/mcc/pkg/product/k0s/util"
	"github.com/Mirantis/mcc/pkg/util"
	"github.com/prometheus/common/log"
)

// Configurer is a generic Ubuntu level configurer implementation. Some of the configurer interface implementation
// might be on OS version specific implementation such as for Bionic.
type Configurer struct {
	configurer.LinuxConfigurer
}

// UploadK0s uploads k0s binary to the host
func (c *Configurer) UploadK0s(version string, k0sConfig *common.GenericHash) error {
	localpath := c.Host.K0sBinary
	if len(localpath) == 0 {
		arch, err := c.Host.ExecWithOutput("uname -m")
		if err != nil {
			return err
		}
		localpath, err = k0sutil.DownloadK0s(version, arch)
		if err != nil {
			return err
		}
	}

	if err := c.Host.Configurer.WriteFileLarge(localpath, "/usr/bin/k0s"); err != nil {
		return err
	}
	return c.Host.Exec(fmt.Sprintf("sudo chmod +x %s", "/usr/bin/k0s"))
}

// WriteFileLarge copies a larger file to the host.
// Use instead of configurer.WriteFile when it seems appropriate
func (c *Configurer) WriteFileLarge(src, dst string) error {
	startTime := time.Now()
	stat, err := os.Stat(src)
	if err != nil {
		return err
	}
	size := stat.Size()

	log.Infof("%s: uploading %s to %s", c.Host, util.FormatBytes(uint64(stat.Size())), dst)

	if err := c.Host.Connection.Upload(src, dst); err != nil {
		return fmt.Errorf("upload failed: %s", err.Error())
	}

	duration := time.Since(startTime).Seconds()
	speed := float64(size) / duration
	log.Infof("%s: transfered %s in %.1f seconds (%s/s)", c.Host, util.FormatBytes(uint64(size)), duration, util.FormatBytes(uint64(speed)))

	return nil
}

// K0sConfigPath returns location of k0s configuration file
func (c *Configurer) K0sConfigPath() string {
	return "/etc/k0s/k0s.yaml"
}

// K0sJoinToken returns location of k0s join token file
func (c *Configurer) K0sJoinToken() string {
	return "/etc/k0s/k0stoken"
}

// InstallBasePackages installs the needed base packages on Ubuntu
func (c *Configurer) InstallBasePackages() error {
	return c.Host.Exec("sudo apt-get update && sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -q curl apt-utils socat iputils-ping")
}

func resolveUbuntuConfigurer(h *api.Host) api.HostConfigurer {
	configurer := &BionicConfigurer{
		Configurer: Configurer{
			LinuxConfigurer: configurer.LinuxConfigurer{
				Host: h,
			},
		},
	}
	return configurer
}

func init() {
	api.RegisterHostConfigurer(resolveUbuntuConfigurer)
}
