package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/Mirantis/mcc/version"

	"github.com/Mirantis/mcc/cmd"

	"github.com/shiena/ansicolor"
	log "github.com/sirupsen/logrus"

	"github.com/urfave/cli/v2"
)

// configureLogger configures log output / formatting
func configureLogger() {
	if runtime.GOOS == "windows" {
		log.SetFormatter(&log.TextFormatter{ForceColors: true})
		log.SetOutput(ansicolor.NewAnsiColorWriter(os.Stdout))
	}
}

func main() {
	configureLogger()

	versionCmd := &cli.Command{
		Name: "version",
		Action: func(ctx *cli.Context) error {
			fmt.Printf("version: %s\n", version.Version)
			fmt.Printf("commit: %s\n", version.GitCommit)
			return nil
		},
	}

	app := &cli.App{
		Name:  "mcc",
		Usage: "Mirantis Cluster Control",
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
			cmd.NewInstallCommand(),
			cmd.RegisterCommand(),
			versionCmd,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}
