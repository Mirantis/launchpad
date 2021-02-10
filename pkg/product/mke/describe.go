package mke

import (
	"os"

	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/phase"
	de "github.com/Mirantis/mcc/pkg/product/mke/phase"
	"gopkg.in/yaml.v2"
)

// Describe - gets information about configured instance
func (p *MKE) Describe(reportName string) error {
	var msr bool
	var mke bool

	if reportName == "msr" {
		msr = true
	}

	if reportName == "mke" {
		mke = true
	}

	if reportName == "config" {
		encoder := yaml.NewEncoder(os.Stdout)
		return encoder.Encode(p.ClusterConfig)
	}

	phaseManager := phase.NewManager(&p.ClusterConfig)
	phaseManager.IgnoreErrors = true

	phaseManager.AddPhases(
		&common.Connect{},
		&de.DetectOS{},
		&de.GatherFacts{},
		&common.Disconnect{},
		&de.Describe{MKE: mke, MSR: msr},
	)

	return phaseManager.Run()
}
