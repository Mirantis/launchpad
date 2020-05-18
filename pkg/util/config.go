package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// ResolveClusterFile resolves the cluster config file based on cli params
func ResolveClusterFile(ctx *cli.Context) ([]byte, error) {
	clusterFile := ctx.String("config")
	fp, err := filepath.Abs(clusterFile)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to lookup current directory name: %v", err)
	}
	file, err := os.Open(fp)
	if err != nil {
		return []byte{}, fmt.Errorf("can not find cluster configuration file: %v", err)
	}
	log.Debugf("opened config file from %s", fp)

	defer file.Close()

	buf, err := ioutil.ReadAll(file)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to read file: %v", err)
	}
	return buf, nil
}
