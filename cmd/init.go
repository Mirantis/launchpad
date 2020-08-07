package cmd

import (
	"os"

	"github.com/Mirantis/mcc/pkg/constant"

	"gopkg.in/yaml.v2"

	"github.com/Mirantis/mcc/pkg/analytics"
	api "github.com/Mirantis/mcc/pkg/apis/v1beta3"
	"github.com/urfave/cli/v2"
)

// NewInitCommand creates new init command to be called from cli
func NewInitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Initialize cluster.yaml file",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "kind",
				Usage:   "What kind of cluster definition we'll create",
				Aliases: []string{"k"},
				Value:   "UCP",
				Hidden:  true, // We don't support anything else than UCP for now
			},
			&cli.BoolFlag{
				Name:    "dtr",
				Usage:   "Init a cluster.yaml file for Docker Trusted Registry (DTR)",
				Aliases: []string{"d"},
				Value:   false,
			},
		},
		Action: func(ctx *cli.Context) error {
			analytics.TrackEvent("Cluster Init Started", nil)
			if err := analytics.RequireRegisteredUser(); err != nil {
				analytics.TrackEvent("Cluster Init Failed", nil)
				return err
			}
			clusterConfig := api.ClusterConfig{
				APIVersion: "launchpad.mirantis.com/v1beta3",
				Kind:       "UCP",
				Metadata: &api.ClusterMeta{
					Name: "my-ucp-cluster",
				},
				Spec: &api.ClusterSpec{
					Engine: api.EngineConfig{
						Version: constant.EngineVersion,
					},
					Ucp: api.UcpConfig{
						Version: constant.UCPVersion,
					},
					Dtr: api.DtrConfig{
						Version:       constant.DTRVersion,
						ReplicaConfig: "sequential",
					},
					Hosts: []*api.Host{
						{
							Address: "10.0.0.1",
							Role:    "manager",
							SSH: &api.SSH{
								User:    "root",
								Port:    22,
								KeyPath: "~/.ssh/id_rsa",
							},
						},
						{
							Address: "10.0.0.2",
							Role:    "worker",
							SSH:     api.DefaultSSH(),
						},
						{
							Address: "10.0.0.3",
							Role:    "dtr",
							SSH:     api.DefaultSSH(),
						},
					},
				},
			}

			encoder := yaml.NewEncoder(os.Stdout)
			err := encoder.Encode(clusterConfig)

			if err != nil {
				analytics.TrackEvent("Cluster Init Failed", nil)
			} else {
				analytics.TrackEvent("Cluster Init Completed", nil)
			}
			return err
		},
	}
}
