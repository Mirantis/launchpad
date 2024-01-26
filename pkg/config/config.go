package config

import (
	"errors"
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
func ProductFromFile(path string) (product.Product, error) { //nolint:ireturn
	data, err := resolveClusterFile(path)
	if err != nil {
		return nil, err
	}
	return ProductFromYAML(data)
}

var errMissingKind = errors.New("configuration does not contain the required keyword 'kind'")

// ProductFromYAML returns a Product from YAML bytes, or an error.
func ProductFromYAML(data []byte) (product.Product, error) { //nolint:ireturn
	config := make(map[string]interface{})
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	if err := migration.Migrate(config); err != nil {
		return nil, fmt.Errorf("failed to migrate configuration: %w", err)
	}

	if config["kind"] == nil {
		return nil, errMissingKind
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal configuration: %w", err)
	}

	plain, err := envsubst.Bytes(data)
	if err != nil {
		return nil, fmt.Errorf("failed to substitute environment variables: %w", err)
	}

	cfg := string(plain)
	if !exec.DisableRedact {
		re := regexp.MustCompile(`(username|password)([:= ]) ?\S+`)
		cfg = re.ReplaceAllString(cfg, "$1$2[REDACTED]")
	}

	log.Debugf("loaded configuration:\n%s", cfg)

	kindStr, ok := config["kind"].(string)
	if !ok {
		return nil, errMissingKind
	}
	switch kindStr {
	case "mke", "mke+msr":
		mke, err := mke.NewMKE(plain)
		if err != nil {
			return nil, fmt.Errorf("failed to parse MKE configuration: %w", err)
		}
		return mke, nil
	default:
		return nil, fmt.Errorf("%w: %s", errUnknownConfigKind, kindStr)
	}
}

var errUnknownConfigKind = errors.New("unknown configuration kind")

// Init returns an example cluster configuration.
func Init(kind string) (interface{}, error) {
	switch kind {
	case "mke", "mke+msr":
		return mke.Init(kind), nil
	default:
		return "", fmt.Errorf("%w: %s", errUnknownConfigKind, kind)
	}
}

var errIsCharDevice = errors.New("is a character device")

func resolveClusterFile(clusterFile string) ([]byte, error) {
	if clusterFile == "-" {
		stat, err := os.Stdin.Stat()
		if err != nil {
			return nil, fmt.Errorf("can't open cluster configuration from stdin: stat: %w", err)
		}
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return nil, fmt.Errorf("error while reading cluster configuration from stdin: %w", err)
			}
			return data, nil
		}

		return nil, fmt.Errorf("can't read from stdin: %w", errIsCharDevice)
	}

	file, err := openClusterFile(clusterFile)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error while reading cluster file %s: %w", clusterFile, err)
	}
	return data, nil
}

var errEmptyFileName = errors.New("empty file name")

func openClusterFile(clusterFile string) (*os.File, error) {
	clusterFileName := detectClusterFile(clusterFile)
	if clusterFileName == "" {
		return nil, fmt.Errorf("can't find cluster configuration file: %w", errEmptyFileName)
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
		return nil, "", fmt.Errorf("failed to lookup current directory name: %w", err)
	}
	file, err = os.Open(fp)
	if err != nil {
		return nil, fp, fmt.Errorf("can not find cluster configuration file: %w", err)
	}
	return file, fp, nil
}
