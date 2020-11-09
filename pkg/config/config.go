package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/a8m/envsubst"
	"gopkg.in/yaml.v2"

	"github.com/Mirantis/mcc/pkg/config/migration"
	// needed to load the migrators
	_ "github.com/Mirantis/mcc/pkg/config/migration/v1beta1"
	// needed to load the migrators
	_ "github.com/Mirantis/mcc/pkg/config/migration/v1beta2"
	// needed to load the migrators
	_ "github.com/Mirantis/mcc/pkg/config/migration/v1beta3"
	// needed to load the migrators
	_ "github.com/Mirantis/mcc/pkg/config/migration/v1"
	"github.com/Mirantis/mcc/pkg/product"
	"github.com/Mirantis/mcc/pkg/product/dockerenterprise"
	log "github.com/sirupsen/logrus"
)

// ProductFromFile loads a yaml file and returns a Product that matches its Kind or an error if the file loading or validation fails
func ProductFromFile(path string) (product.Product, error) {
	data, err := resolveClusterFile(path)
	if err != nil {
		return nil, err
	}
	data, err = envsubst.Bytes(data)
	if err != nil {
		return nil, err
	}

	return productFromYAML(data)
}

func productFromYAML(data []byte) (product.Product, error) {
	c := make(map[string]interface{})
	if err := yaml.Unmarshal(data, c); err != nil {
		return nil, err
	}

	if err := migration.Migrate(c); err != nil {
		return nil, err
	}

	if c["kind"] == nil {
		return nil, fmt.Errorf("configuration does not contain the required keyword 'kind'")
	}

	plain, err := yaml.Marshal(c)
	if err != nil {
		return nil, err
	}

	log.Debugf("loaded configuration:\n%s", plain)

	switch c["kind"].(string) {
	case "DockerEnterprise":
		return dockerenterprise.NewDockerEnterprise(plain)
	default:
		return nil, fmt.Errorf("unknown configuration kind '%s'", c["kind"].(string))
	}
}

// Init returns an example cluster configuration
func Init(kind string) (interface{}, error) {
	switch kind {
	case "DockerEnterprise":
		return dockerenterprise.Init(), nil
	default:
		return "", fmt.Errorf("unknown configuration kind '%s'", kind)
	}
}

func resolveClusterFile(clusterFile string) ([]byte, error) {
	if clusterFile == "-" {
		stat, err := os.Stdin.Stat()
		if err == nil {
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				return ioutil.ReadAll(os.Stdin)
			}
		}

		return nil, fmt.Errorf("can't open cluster configuration from stdin")
	}

	file, err := openClusterFile(clusterFile)
	defer file.Close()
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(file)
}

func openClusterFile(clusterFile string) (*os.File, error) {
	clusterFileName := detectClusterFile(clusterFile)
	if clusterFileName == "" {
		return nil, fmt.Errorf("can't find cluster configuration file %s", clusterFile)
	}

	file, fp, err := openFile(clusterFileName)
	if err != nil {
		return nil, fmt.Errorf("error while opening cluster file %s: %w", clusterFileName, err)
	}

	log.Debugf("opened config file from %s", fp)
	return file, nil
}

func detectClusterFile(clusterFile string) string {
	// the first option always is the file name provided by the user
	possibleOptions := []string{clusterFile}
	if strings.HasSuffix(clusterFile, ".yaml") {
		possibleOptions = append(possibleOptions, strings.ReplaceAll(clusterFile, ".yaml", ".yml"))
	}
	if strings.HasSuffix(clusterFile, ".yml") {
		possibleOptions = append(possibleOptions, strings.ReplaceAll(clusterFile, ".yml", ".yaml"))
	}

	for _, option := range possibleOptions {
		if _, err := os.Stat(option); err != nil {
			continue
		}

		return option
	}

	return ""
}

func openFile(fileName string) (file *os.File, path string, err error) {
	fp, err := filepath.Abs(fileName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to lookup current directory name: %v", err)
	}
	file, err = os.Open(fp)
	if err != nil {
		return nil, fp, fmt.Errorf("can not find cluster configuration file: %v", err)
	}
	return file, fp, nil
}
