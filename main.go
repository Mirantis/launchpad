package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Mirantis/mcc/version"

	"github.com/Mirantis/mcc/cmd"

	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/writer"
	"github.com/urfave/cli/v2"
)

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
		Commands: []*cli.Command{
			cmd.NewInstallCommand(),
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
	stdoutWriter := &writer.Hook{
		Writer: os.Stdout,
		LogLevels: []log.Level{
			log.InfoLevel,
			log.PanicLevel,
			log.FatalLevel,
			log.ErrorLevel,
			log.WarnLevel,
		},
	}
	// Add debug level to stdout hook if set by user
	if ctx.Bool("debug") {
		stdoutWriter.LogLevels = append([]log.Level{log.DebugLevel}, stdoutWriter.LogLevels...)
	}
	// stdout hook on by default of course
	log.AddHook(stdoutWriter)
}
