package bundle

import (
	api "github.com/Mirantis/mcc/pkg/apis/v1beta2"
	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/phase"
)

// Download downloads a UCP client bundle
func Download(clusterFile string, username string, password string) error {
	cfgData, err := config.ResolveClusterFile(clusterFile)
	if err != nil {
		return err
	}
	clusterConfig, err := config.FromYaml(cfgData)
	if err != nil {
		return err
	}

	if err = config.Validate(&clusterConfig); err != nil {
		return err
	}

	manager := clusterConfig.Spec.Managers()[0]
	newHosts := make(api.Hosts, 1)
	newHosts[0] = manager
	clusterConfig.Spec.Hosts = newHosts

	phaseManager := phase.NewManager(&clusterConfig)
	phaseManager.AddPhase(&phase.Connect{})
	phaseManager.AddPhase(&phase.GatherFacts{})
	phaseManager.AddPhase(&phase.ValidateHosts{})
	phaseManager.AddPhase(&phase.DownloadBundle{Username: username, Password: password})
	phaseManager.AddPhase(&phase.Disconnect{})

	if err = phaseManager.Run(); err != nil {
		return err
	}

	return nil
}
