package main

import (
	"os"

	"github.com/Mirantis/mcc/pkg/install"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func main() {
	installCmd := &cli.Command{
		Name:  "install",
		Usage: "Install a new cluster",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Usage:   "Path to cluster config yaml",
				Aliases: []string{"c"},
				Value:   "cluster.yaml",
			},
		},
		Action: func(ctx *cli.Context) error {
			log.Info("Install called")
			log.Debug("Debug is on.....")

			err := install.Install(ctx)
			return err
		},
	}

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "debug",
				Usage:   "Enable debug logging",
				Aliases: []string{"d"},
				EnvVars: []string{"DEBUG"},
			},
		},
		Before: func(ctx *cli.Context) error {
			if ctx.Bool("debug") {
				log.SetLevel(log.DebugLevel)
			}
			return nil
		},
		Commands: []*cli.Command{
			installCmd,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}
