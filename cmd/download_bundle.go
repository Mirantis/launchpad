package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/cmd/bundle"
	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/ssh/terminal"
)

// NewDownloadBundleCommand creates a download bundle command to be called via the CLI
func NewDownloadBundleCommand() *cli.Command {
	return &cli.Command{
		Name:  "download-bundle",
		Usage: "Download a client bundle",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "username",
				Usage:   "Username",
				Aliases: []string{"u"},
			},
			&cli.StringFlag{
				Name:    "password",
				Usage:   "Password",
				Aliases: []string{"p"},
			},
			&cli.StringFlag{
				Name:    "config",
				Usage:   "Path to cluster config yaml",
				Aliases: []string{"c"},
				Value:   "cluster.yaml",
			},
		},
		Action: func(ctx *cli.Context) error {
			username, password := resolveCredentials(ctx)

			err := bundle.Download(ctx.String("config"), username, password)
			if err != nil {
				analytics.TrackEvent("Bundle Download Failed", nil)
			} else {
				analytics.TrackEvent("Bundle Download Completed", nil)
			}

			return err
		},
	}
}

func resolveCredentials(ctx *cli.Context) (username, password string) {
	username = ctx.String("username")
	if username == "" {
		username = readUsernameFrom(os.Stdin)
	}
	password = ctx.String("password")
	if password == "" {
		password = readPasswordFrom(os.Stdin)
	}
	return username, password
}

func readPasswordFrom(in io.Reader) string {
	if in == os.Stdin {
		fd := int(os.Stdin.Fd())
		if terminal.IsTerminal(fd) {
			fmt.Fprint(os.Stderr, "Password: ")
			pw, err := terminal.ReadPassword(fd)
			fmt.Fprintln(os.Stderr)
			if err != nil {
				fmt.Println("error while reading password: ", err)
			}
			return string(pw)
		}
	}
	fmt.Printf("Password: ")
	return readFrom(in)
}

func readUsernameFrom(in io.Reader) string {
	fmt.Printf("Username: ")
	return readFrom(in)
}

func readFrom(in io.Reader) string {
	reader := bufio.NewReader(in)
	line, _, err := reader.ReadLine()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	return string(line)
}
