package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

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
