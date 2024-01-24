package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Mirantis/mcc/pkg/config/migration"
	// needed to load the migrators.
	_ "github.com/Mirantis/mcc/pkg/config/migration/v1"
	// needed to load the migrators.
	_ "github.com/Mirantis/mcc/pkg/config/migration/v11"
	// needed to load the migrators.
	_ "github.com/Mirantis/mcc/pkg/config/migration/v12"
	// needed to load the migrators.
	_ "github.com/Mirantis/mcc/pkg/config/migration/v13"
	// needed to load the migrators.
	_ "github.com/Mirantis/mcc/pkg/config/migration/v1beta1"
	// needed to load the migrators.
	_ "github.com/Mirantis/mcc/pkg/config/migration/v1beta2"
	// needed to load the migrators.
	_ "github.com/Mirantis/mcc/pkg/config/migration/v1beta3"
	"github.com/Mirantis/mcc/pkg/product"
	"github.com/Mirantis/mcc/pkg/product/mke"
	"github.com/a8m/envsubst"
	"github.com/k0sproject/rig/exec"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// ProductFromFile loads a yaml file and returns a Product that matches its Kind or an error if the file loading or validation fails.
func ProductFromFile(path string) (product.Product, error) {
	data, err := resolveClusterFile(path)
	if err != nil {
		return nil, err
	}
	return ProductFromYAML(data)
}

// ProductFromYAML returns a Product from YAML bytes, or an error.
func ProductFromYAML(data []byte) (product.Product, error) {
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

	data, err := yaml.Marshal(c)
	if err != nil {
		return nil, err
	}

	plain, err := envsubst.Bytes(data)
	if err != nil {
		return nil, err
	}

	cfg := string(plain)
	if !exec.DisableRedact {
		re := regexp.MustCompile(`(username|password)([:= ]) ?\S+`)
		cfg = re.ReplaceAllString(cfg, "$1$2[REDACTED]")
	}

	log.Debugf("loaded configuration:\n%s", cfg)

	switch c["kind"].(string) {
	case "mke", "mke+msr":
		return mke.NewMKE(plain)
	default:
		return nil, fmt.Errorf("unknown configuration kind '%s'", c["kind"].(string))
	}
}

// Init returns an example cluster configuration.
func Init(kind string) (interface{}, error) {
	switch kind {
	case "mke", "mke+msr":
		return mke.Init(kind), nil
	default:
		return "", fmt.Errorf("unknown configuration kind '%s'", kind)
	}
}

func resolveClusterFile(clusterFile string) ([]byte, error) {
	if clusterFile == "-" {
		stat, err := os.Stdin.Stat()
		if err == nil {
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				return io.ReadAll(os.Stdin)
			}
		}

		return nil, fmt.Errorf("can't open cluster configuration from stdin")
	}

	file, err := openClusterFile(clusterFile)
	defer file.Close() //nolint:staticcheck
	if err != nil {
		return nil, err
	}

	return io.ReadAll(file)
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
