package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta1"
	validator "github.com/go-playground/validator/v10"
	log "github.com/sirupsen/logrus"
)

// FromYaml loads the cluster config from given yaml data
func FromYaml(data []byte) (api.ClusterConfig, error) {
	c := api.ClusterConfig{}

	err := yaml.Unmarshal(data, &c)
	if err != nil {
		return c, err
	}

	return c, nil
}

// Validate validates that everything in the config makes sense
// Currently we do only very "static" validation using https://github.com/go-playground/validator
func Validate(c *api.ClusterConfig) error {
	validator := validator.New()
	return validator.Struct(c)
}

// ResolveClusterFile looks for the cluster.yaml file, based on the value.
// It returns the contents of this file as []byte if found,
// or error if it didn't.
func ResolveClusterFile(clusterFile string) ([]byte, error) {
	file, err := openClusterFile(clusterFile)
	defer file.Close()

	buf, err := ioutil.ReadAll(file)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to read file: %v", err)
	}
	return buf, nil
}

func openClusterFile(clusterFile string) (*os.File, error) {
	file, fp, err := openFileWithName(clusterFile)
	if err != nil {
		if strings.HasSuffix(clusterFile, ".yaml") {
			clusterFile = strings.ReplaceAll(clusterFile, ".yaml", ".yml")
			var newError error
			file, fp, newError = openFileWithName(clusterFile)
			if newError != nil {
				return nil, fmt.Errorf("can not find cluster configuration file: %v: %v", newError, err)
			}
		} else if strings.HasSuffix(clusterFile, ".yml") {
			clusterFile = strings.ReplaceAll(clusterFile, ".yml", ".yaml")
			var newError error
			file, fp, newError = openFileWithName(clusterFile)
			if newError != nil {
				return nil, fmt.Errorf("can not find cluster configuration file: %v: %v", newError, err)
			}
		}
	}
	log.Debugf("opened config file from %s", fp)

	return file, nil
}

func openFileWithName(fileName string) (file *os.File, path string, err error) {
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
