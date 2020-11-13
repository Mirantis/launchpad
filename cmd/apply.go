package cmd

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/exec"
	mcclog "github.com/Mirantis/mcc/pkg/log"
	"github.com/Mirantis/mcc/pkg/util"
	"github.com/Mirantis/mcc/version"

	"github.com/mattn/go-isatty"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	event "gopkg.in/segmentio/analytics-go.v3"
)

// NewApplyCommand creates new apply command to be called from cli
func NewApplyCommand() *cli.Command {
	return &cli.Command{
		Name:  "apply",
		Usage: "Apply a cluster configuration",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:      "config",
				Usage:     "Path to cluster config yaml. Use '-' to read from stdin.",
				Aliases:   []string{"c"},
				Value:     "launchpad.yaml",
				TakesFile: true,
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Allow continuing in some situations where prerequisite checks fail",
				Value:   false,
			},
			&cli.BoolFlag{
				Name:   "disable-cleanup",
				Usage:  "Do not perform cleanup on failure",
				Value:  false,
				Hidden: true,
			},
			&cli.BoolFlag{
				Name:  "confirm",
				Usage: "Ask confirmation for all commands",
				Value: false,
			},
			&cli.BoolFlag{
				Name:  "disable-redact",
				Usage: "Do not hide sensitive information in the output",
				Value: false,
			},
		},
		Before: func(ctx *cli.Context) error {
			exec.Confirm = ctx.Bool("confirm")
			exec.DisableRedact = ctx.Bool("disable-redact")
			if !ctx.Bool("accept-license") {
				return analytics.RequireRegisteredUser()
			}
			return nil
		},
		Action: func(ctx *cli.Context) error {
			var (
				logFile *os.File
				err     error
			)

			start := time.Now()
			analytics.TrackEvent("Cluster Apply Started", nil)

			product, err := config.ProductFromFile(ctx.String("config"))
			if err != nil {
				return err
			}

			defer func() {
				if err != nil && logFile != nil {
					log.Infof("See %s for more logs ", logFile.Name())
				}

			}()

			// Add logger to dump all log levels to file
			logFile, err = addFileLogger(product.ClusterName())
			if err != nil {
				return err
			}

			if isatty.IsTerminal(os.Stdout.Fd()) {
				os.Stdout.WriteString(util.Logo)
				os.Stdout.WriteString(fmt.Sprintf("   Mirantis Launchpad (c) 2020 Mirantis, Inc.                          v%s\n\n", version.Version))
			}

			err = product.Apply(ctx.Bool("disable-cleanup"), ctx.Bool("force"))

			if err != nil {
				analytics.TrackEvent("Cluster Apply Failed", nil)
			} else {
				duration := time.Since(start)
				props := event.Properties{
					"duration": duration.Seconds(),
				}
				analytics.TrackEvent("Cluster Apply Completed", props)
			}

			return err
		},
	}
}

const fileMode = 0700

func addFileLogger(clusterName string) (*os.File, error) {
	home, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	clusterDir := path.Join(home, constant.StateBaseDir, "cluster", clusterName)
	if err := util.EnsureDir(clusterDir); err != nil {
		return nil, fmt.Errorf("error while creating directory for apply logs: %w", err)
	}
	logFileName := path.Join(clusterDir, "apply.log")
	logFile, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fileMode)

	if err != nil {
		return nil, fmt.Errorf("Failed to create apply log at %s: %s", logFileName, err.Error())
	}

	// Send all logs to named file, this ensures we always have debug logs also available when needed
	log.AddHook(mcclog.NewFileHook(logFile))

	return logFile, nil
}
