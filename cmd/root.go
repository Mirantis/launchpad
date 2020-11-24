package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/Mirantis/mcc/pkg/analytics"
	mcclog "github.com/Mirantis/mcc/pkg/log"
	"github.com/Mirantis/mcc/version"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	rootCmd = &cobra.Command{
		Use:   "mcc",
		Short: "Mirantis Launchpad",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// You can bind cobra and viper in a few locations, but PersistencePreRunE on the root command works well
			initLogger()
			initAnalytics(telemetry)
			return initializeConfig(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("blah")
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			closeAnalyticsClient()
			if downloadLaunchpad {
				latest := <-upgradeChan
				if latest != nil {
					println(fmt.Sprintf("\nA new version (%s) of `launchpad` is available. Please visit %s or run `launchpad download-launchpad` to upgrade the tool.", latest.TagName, latest.URL))
				}
			}
			return nil
		},
	}

	debug, telemetry, checkUpgrade, acceptLicence, downloadLaunchpad bool
	upgradeChan                                                      = make(chan *version.LaunchpadRelease)
)

func init() {
	upgradeChan = make(chan *version.LaunchpadRelease)

	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVarP(&telemetry, "disable-telemetry", "t", false, "Disable telemetry")
	rootCmd.PersistentFlags().BoolVarP(&checkUpgrade, "disable-upgrade-check", "u", false, "Disable upgrade check")
	rootCmd.PersistentFlags().BoolVarP(&acceptLicence, "accept-license", "a", false, "Accept License Agreement: https://github.com/Mirantis/launchpad/blob/master/LICENSE")
	rootCmd.PersistentFlags().BoolVarP(&downloadLaunchpad, "download-launchpad", "", false, "Download launchpad")
}

func initializeConfig(cmd *cobra.Command) error {
	v := viper.New()

	// Attempt to read the config file, gracefully ignoring errors
	// caused by a config file not being found. Return an error
	// if we cannot parse the config file.
	if err := v.ReadInConfig(); err != nil {
		// It's okay if there isn't a config file
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	// When we bind flags to environment variables expect that the
	// environment variables are prefixed, e.g. a flag like --number
	// binds to an environment variable STING_NUMBER. This helps
	// avoid conflicts.
	v.SetEnvPrefix("")

	// Bind to environment variables
	// Works great for simple config names, but needs help for names
	// like --favorite-color which we fix in the bindFlags function
	v.AutomaticEnv()

	// Bind the current command's flags to viper
	bindFlags(cmd, v)

	return nil
}

func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Environment variables can't have dashes in them, so bind them to their equivalent
		// keys with underscores, e.g. --favorite-color to STING_FAVORITE_COLOR
		if strings.Contains(f.Name, "-") {
			envVarSuffix := strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
			v.BindEnv(f.Name, fmt.Sprintf("%s_%s", "", envVarSuffix))
		}

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && v.IsSet(f.Name) {
			val := v.Get(f.Name)
			cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
}
func initLogger() {
	log.SetLevel(log.TraceLevel)
	log.SetOutput(ioutil.Discard) // Send all logs to nowhere by default

	// Send logs with level >= INFO to stdout

	// stdout hook on by default of course
	log.AddHook(mcclog.NewStdoutHook())
}

func initAnalytics(disable bool) {
	analytics.Enabled(disable)
}

func closeAnalyticsClient() {
	if err := analytics.Close(); err != nil {
		log.Debugf("Error while closing analytics client: %v", err)
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		log.Fatal(err)
	}
}
