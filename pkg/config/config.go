package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/Mirantis/mcc/pkg/apis/v1beta1"
	api "github.com/Mirantis/mcc/pkg/apis/v1beta2"
	validator "github.com/go-playground/validator/v10"
	log "github.com/sirupsen/logrus"
)

// Version is used for determining the configuration file type and version
type Version struct {
	APIVersion string `yaml:"apiVersion" validate:"required,gt=2"`
	Kind       string `yaml:"kind" validate:"required,gt=2"`
}

// FromYaml loads the cluster config from given yaml data
func FromYaml(data []byte) (api.ClusterConfig, error) {
	c := api.ClusterConfig{}

	cv := Version{}
	err := yaml.Unmarshal(data, &cv)
	if err != nil {
		return c, err
	}

	if cv.Kind != "UCP" {
		return c, fmt.Errorf("Unknown kind: %s", cv.Kind)
	}

	if cv.APIVersion == "launchpad.mirantis.com/v1beta1" {
		v1beta1.MigrateV1Beta2(&data)
	}

	err = yaml.Unmarshal(data, &c)
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
	clusterFileName := detectClusterFile(clusterFile)
	if clusterFileName == "" {
		return nil, fmt.Errorf("can not find cluster configuration file %s", clusterFile)
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
