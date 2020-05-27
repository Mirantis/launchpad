package cmd

import (
	"os"

	"github.com/Mirantis/mcc/pkg/constant"

	"gopkg.in/yaml.v2"

	"github.com/Mirantis/mcc/pkg/analytics"
	api "github.com/Mirantis/mcc/pkg/apis/v1beta1"
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
		},
		Action: func(ctx *cli.Context) error {

			clusterConfig := api.ClusterConfig{
				APIVersion: "launchpad.mirantis.com/v1beta1",
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
					Hosts: []*api.Host{
						&api.Host{
							Address:    "1.2.3.4",
							Role:       "manager",
							SSHPort:    22,
							SSHKeyPath: "~/.ssh/id_rsa",
							User:       "root",
						},
						&api.Host{
							Address:    "4.5.6.7",
							Role:       "worker",
							SSHPort:    22,
							SSHKeyPath: "~/.ssh/id_rsa",
							User:       "root",
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
