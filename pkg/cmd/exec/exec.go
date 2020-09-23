package exec

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/config"

	log "github.com/sirupsen/logrus"
)

// Exec ...
func Exec(configFile string, address, cmd string) error {
	cfgData, err := config.ResolveClusterFile(configFile)
	if err != nil {
		return err
	}
	clusterConfig, err := config.FromYaml(cfgData)
	if err != nil {
		return err
	}

	if err := config.Validate(&clusterConfig); err != nil {
		return err
	}

	// if isatty.IsTerminal(os.Stdout.Fd()) {
	// 	os.Stdout.WriteString(util.Logo)
	// 	os.Stdout.WriteString(fmt.Sprintf("   Mirantis Launchpad (c) 2020 Mirantis, Inc.                          v%s\n\n", version.Version))
	// }

	log.Debugf("loaded cluster cfg: %+v", clusterConfig)

	host := clusterConfig.Spec.Hosts.Find(func(h *api.Host) bool { return h.Address == address })
	if host == nil {
		return fmt.Errorf("Host with address %s not found in configuration", address)
	}

	err = host.Connect()
	if err != nil {
		println(fmt.Sprintf("Failed to connect: %s", err.Error()))
		return err
	}

	if cmd == "" {
		return host.Connection.ExecInteractive()
	}

	return host.ExecCmd(cmd, "", true, false)
}
