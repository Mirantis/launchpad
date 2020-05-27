package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Mirantis/mcc/cmd"
	mcclog "github.com/Mirantis/mcc/pkg/log"
	"github.com/Mirantis/mcc/version"
	log "github.com/sirupsen/logrus"

	"github.com/urfave/cli/v2"
)

func init() {
	log.SetOutput(os.Stdout)
}

func main() {
	versionCmd := &cli.Command{
		Name: "version",
		Action: func(ctx *cli.Context) error {
			fmt.Printf("version: %s\n", version.Version)
			fmt.Printf("commit: %s\n", version.GitCommit)
			return nil
		},
	}

	app := &cli.App{
		Name:                 "launchpad",
		Usage:                "Mirantis Launchpad",
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "debug",
				Usage:   "Enable debug logging",
				Aliases: []string{"d"},
				EnvVars: []string{"DEBUG"},
			},
		},
		Before: func(ctx *cli.Context) error {
			initLogger(ctx)
			return nil
		},
		After: func(c *cli.Context) error {
			version.CheckForUpgrade()
			return nil
		},
		Commands: []*cli.Command{
			cmd.NewApplyCommand(),
			cmd.RegisterCommand(),
			cmd.NewDownloadBundleCommand(),
			versionCmd,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}

func initLogger(ctx *cli.Context) {
	// Enable debug logging always, we'll setup hooks later to direct logs based on level
	log.SetLevel(log.DebugLevel)
	log.SetOutput(ioutil.Discard) // Send all logs to nowhere by default

	// Send logs with level >= INFO to stdout

	// stdout hook on by default of course
	log.AddHook(mcclog.NewStdoutHook(ctx.Bool("debug")))
}
