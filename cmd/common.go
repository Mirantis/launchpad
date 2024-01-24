package cmd

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/constant"
	mcclog "github.com/Mirantis/mcc/pkg/log"
	"github.com/Mirantis/mcc/pkg/product/mke/phase"
	"github.com/Mirantis/mcc/pkg/util"
	"github.com/Mirantis/mcc/version"
	"github.com/k0sproject/rig"
	"github.com/k0sproject/rig/exec"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var (
	debugFlag = &cli.BoolFlag{
		Name:    "debug",
		Usage:   "Enable debug logging",
		Aliases: []string{"d"},
		EnvVars: []string{"DEBUG"},
	}

	traceFlag = &cli.BoolFlag{
		Name:    "trace",
		Usage:   "Enable trace logging",
		Aliases: []string{"dd"},
		EnvVars: []string{"TRACE"},
		Hidden:  true,
	}

	telemetryFlag = &cli.BoolFlag{
		Name:    "disable-telemetry",
		Usage:   "Disable telemetry",
		EnvVars: []string{"DISABLE_TELEMETRY"},
	}

	licenseFlag = &cli.BoolFlag{
		Name:    "accept-license",
		Usage:   "Accept License Agreement: https://github.com/Mirantis/launchpad/blob/master/LICENSE",
		Aliases: []string{"a"},
		EnvVars: []string{"ACCEPT_LICENSE"},
	}

	upgradeFlag = &cli.BoolFlag{
		Name:    "disable-upgrade-check",
		Usage:   "Disable upgrade check",
		Aliases: []string{"u"},
		EnvVars: []string{"DISABLE_UPGRADE_CHECK"},
	}

	configFlag = &cli.StringFlag{
		Name:      "config",
		Usage:     "Path to cluster config yaml. Use '-' to read from stdin.",
		Aliases:   []string{"c"},
		Value:     "launchpad.yaml",
		TakesFile: true,
	}

	confirmFlag = &cli.BoolFlag{
		Name:  "confirm",
		Usage: "Ask confirmation for all commands",
		Value: false,
	}

	redactFlag = &cli.BoolFlag{
		Name:  "disable-redact",
		Usage: "Do not hide sensitive information in the output",
		Value: false,
	}

	// GlobalFlags is a set of flags to be included in most commands.
	GlobalFlags = []cli.Flag{
		debugFlag,
		traceFlag,
		telemetryFlag,
		licenseFlag,
		upgradeFlag,
	}

	upgradeChan = make(chan *version.LaunchpadRelease)
)

// actions can be used to chain action functions (for urfave/cli's Before, After, etc).
func actions(funcs ...func(*cli.Context) error) func(*cli.Context) error {
	return func(ctx *cli.Context) error {
		for _, f := range funcs {
			if err := f(ctx); err != nil {
				return err
			}
		}
		return nil
	}
}

func startUpgradeCheck(ctx *cli.Context) error {
	phase.DisableUpgradeCheck = ctx.Bool("disable-upgrade-check")

	go func() {
		if ctx.Command.Name != "download-launchpad" && version.IsProduction() && !ctx.Bool("disable-upgrade-check") {
			upgradeChan <- version.GetUpgrade()
		} else {
			upgradeChan <- nil
		}
	}()

	return nil
}

func initLogger(ctx *cli.Context) error {
	mcclog.Debug = ctx.Bool("debug") || ctx.Bool("trace")
	mcclog.Trace = ctx.Bool("trace")
	log.SetLevel(log.TraceLevel)
	log.SetOutput(io.Discard) // Send all logs to nowhere by default.

	// Send logs with level >= INFO to stdout.

	// stdout hook on by default of course.
	log.AddHook(mcclog.NewStdoutHook())
	rig.SetLogger(log.StandardLogger())

	return nil
}

func initAnalytics(ctx *cli.Context) error {
	if ctx.Bool("disable-telemetry") {
		analytics.Enabled(false)
	}
	return nil
}

func closeAnalytics(ctx *cli.Context) error {
	if !ctx.Bool("disable-telemetry") {
		if err := analytics.Close(); err != nil {
			log.Debugf("Error while closing analytics client: %v", err)
		}
	}
	return nil
}

func upgradeCheckResult(ctx *cli.Context) error {
	if ctx.Command.Name != "download-launchpad" {
		latest := <-upgradeChan
		if latest != nil {
			println(fmt.Sprintf("\nA new version (%s) of `launchpad` is available. Please visit %s or run `launchpad download-launchpad` to upgrade the tool.", latest.TagName, latest.URL))
		}
	}
	return nil
}

func initExec(ctx *cli.Context) error {
	exec.Confirm = ctx.Bool("confirm")
	exec.DisableRedact = ctx.Bool("disable-redact")
	return nil
}

func checkLicense(ctx *cli.Context) error {
	if !ctx.Bool("accept-license") {
		return analytics.RequireRegisteredUser()
	}
	return nil
}

func addFileLogger(clusterName, filename string) (*os.File, error) {
	home, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	clusterDir := path.Join(home, constant.StateBaseDir, "cluster", clusterName)
	if err := util.EnsureDir(clusterDir); err != nil {
		return nil, fmt.Errorf("error while creating directory for logs: %w", err)
	}
	logFileName := path.Join(clusterDir, filename)
	logFile, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, fmt.Errorf("Failed to create log file at %s: %s", logFileName, err.Error())
	}

	// Send all logs to named file, this ensures we always have debug logs also available when needed.
	log.AddHook(mcclog.NewFileHook(logFile))

	return logFile, nil
}
