package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/cmd/containercloud"
	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/util"
	"github.com/Mirantis/mcc/version"
	"github.com/mattn/go-isatty"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	event "gopkg.in/segmentio/analytics-go.v3"
)

var ctx *cli.Context

// ContainerCloudCommand creates the cli command to download the
// bootstrap files for the Docker Enterprise Container Cloud.
func ContainerCloudCommand() *cli.Command {
	return &cli.Command{
		Name:        "container-cloud",
		Usage:       "Download bootstrap bundle for the Docker Enterprise Container Cloud",
		Description: "Download bootstrap bundle for the Docker Enterprise Container Cloud",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "release",
				Usage: "Container Cloud KaaSRelease file",
			},
			&cli.StringFlag{
				Name:  "cdn",
				Usage: "Name of CDN region",
			},
			&cli.StringFlag{
				Name:  "cdn-base-url",
				Usage: "Base URL for artifacts",
			},
			&cli.StringFlag{
				Name:  "releases-base-url",
				Usage: "Base URL for releases metadata files",
			},
			&cli.StringFlag{
				Name:  "cluster-releases-dir",
				Usage: "Directory to look for Cluster releases",
			},
			&cli.StringFlag{
				Name:  "dir",
				Usage: "Output dir for Container Cloud bootstrap files",
			},
		},
		/*		Before: func(c *cli.Context) error {
				if !c.Bool("accept-license") {
					return analytics.RequireRegisteredUser()
				}
				return nil
			},*/
		Action: func(c *cli.Context) error {
			ctx = c
			start := time.Now()
			analytics.TrackEvent("Container Cloud Download Started", nil)
			err := initLogger()
			if err != nil {
				return err
			}
			targetDir := getTargetDir()
			region := getCDNRegion()
			baseURL, err := getBaseURL()
			if err != nil {
				return err
			}
			releaseFile := getKaaSReleaseFilePath()
			releasesBaseURL := getReleasesBaseURL(baseURL)
			clusterReleasesDir := getClusterReleasesDir()
			d := &containercloud.DownloadBootstrapBundle{
				TargetDir:          targetDir,
				Region:             region,
				BaseURL:            baseURL,
				ReleaseFile:        releaseFile,
				ReleasesBaseURL:    releasesBaseURL,
				ClusterReleasesDir: clusterReleasesDir,
			}
			err = d.Init()
			if err != nil {
				return err
			}
			err = d.Run()
			if err != nil {
				return err
			}
			if err != nil {
				analytics.TrackEvent("Container Cloud Download Failed", nil)
			} else {
				duration := time.Since(start)
				props := event.Properties{
					"duration": duration.Seconds(),
				}
				analytics.TrackEvent("Container Cloud Download Completed", props)
			}
			return err
		},
	}
}

// Helper function to get the relative path to target directory from the
// command line flag or from environment var.
// Default target directory is 'kaas-bootstrap' in the current dir.
func getTargetDir() (dir string) {
	if dir = ctx.String("dir"); dir != "" {
		log.Debugf("Set target dir from CLI flag: %v\n", dir)
		return dir
	}
	if dir = os.Getenv(constant.TargetDirEnvVar); dir != "" {
		log.Debugf("Set target dir from env var %s: %v\n", constant.TargetDirEnvVar, dir)
		return dir
	}
	log.Debugf("Set target dir to the default: %v\n", constant.DefaultTargetDir)
	return constant.DefaultTargetDir
}

// Retrieve the CDN region name from cli flag, env var or constant default,
// in the order of precedence.
func getCDNRegion() string {
	if c := ctx.String("cdn"); c != "" {
		log.Debugf("Using CDN region from CLI flag: %s\n", c)
		return c
	}
	if c := os.Getenv(constant.KaaSCDNRegionEnvVar); c != "" {
		log.Debugf("Using CDN region from %s env var: %s\n", constant.KaaSCDNRegionEnvVar, c)
		return c
	}
	log.Debugf("Using CDN region from defaults: %s\n", constant.DefaultCDNRegion)
	return constant.DefaultCDNRegion
}

// Get the base URL for accessing artifacts from the cli flag, env var,
// or generate from constant based on the name of the CDN region.
func getBaseURL() (string, error) {
	if c := ctx.String("cdn-base-url"); c != "" {
		log.Debugf("Using base CDN URL from CLI flag: %s\n", c)
		return c, nil
	}
	if c := os.Getenv(constant.KaaSCDNBaseURLEnvVar); c != "" {
		log.Debugf("Using base CDN URL from %s env var: %s", constant.KaaSCDNBaseURLEnvVar, c)
		return c, nil
	}
	region := getCDNRegion()
	switch region {
	case "internal-ci":
		return constant.InternalCdnBaseURL, nil
	case "internal-eu":
		return constant.InternalEuCdnBaseURL, nil
	case "public-ci":
		return constant.PublicCICdnBaseURL, nil
	case "public":
		return constant.PublicCdnBaseURL, nil
	default:
		err := fmt.Errorf("Unknown CDN region: %s", region)
		return "", err
	}
}

// Get the base URL to download KaaSReleas and ClusterRelease files from.
func getReleasesBaseURL(baseURL string) string {
	if u := ctx.String("release-base-url"); u != "" {
		return u
	}
	if u := os.Getenv(constant.KaaSReleasesBaseURLEnvVar); u != "" {
		return u
	}
	u := fmt.Sprintf("%s/%s", baseURL, constant.DefaultReleasesPath)
	return u
}

// Get the name of the directory that contains the ClusterRelease objects.
func getClusterReleasesDir() string {
	if s := ctx.String("cluster-releases-dir"); s != "" {
		return s
	}
	if s := os.Getenv(constant.ClusterReleasesDirEnvVar); s != "" {
		return s
	}
	return ""
}

// Get the path to the file that contains KaaSRelease. Use the cli flag or env var,
// in order of precedence. If not specified, the file must be downloaded from the
// known location first.
func getKaaSReleaseFilePath() string {
	if r := ctx.String("release"); r != "" {
		log.Debugf("Setting release file path from CLI flag: %s\n", r)
		return r
	}
	if r := os.Getenv(constant.KaaSReleasesYamlEnvVar); r != "" {
		log.Debugf("Setting release file path from env var %s: %s\n", constant.KaaSReleasesYamlEnvVar, r)
		return r
	}
	log.Printf("No release file path given, will download the latest release\n")
	return ""
}

func initLogger() error {
	if isatty.IsTerminal(os.Stdout.Fd()) {
		os.Stdout.WriteString(util.Logo)
		os.Stdout.WriteString(fmt.Sprintf("   Mirantis Launchpad (c) 2020 Mirantis, Inc.                          v%s\n\n", version.Version))
	}
	return nil
}
