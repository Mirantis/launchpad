package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Mirantis/launchpad/pkg/config/migration"
	// needed to load the migrators.
	_ "github.com/Mirantis/launchpad/pkg/config/migration/v1"
	// needed to load the migrators.
	_ "github.com/Mirantis/launchpad/pkg/config/migration/v11"
	// needed to load the migrators.
	_ "github.com/Mirantis/launchpad/pkg/config/migration/v12"
	// needed to load the migrators.
	_ "github.com/Mirantis/launchpad/pkg/config/migration/v13"
	// needed to load the migrators.
	_ "github.com/Mirantis/launchpad/pkg/config/migration/v14"
	// needed to load the migrators.
	_ "github.com/Mirantis/launchpad/pkg/config/migration/v15"
	// needed to load the migrators.
	_ "github.com/Mirantis/launchpad/pkg/config/migration/v1beta1"
	// needed to load the migrators.
	_ "github.com/Mirantis/launchpad/pkg/config/migration/v1beta2"
	// needed to load the migrators.
	_ "github.com/Mirantis/launchpad/pkg/config/migration/v1beta3"
	"github.com/Mirantis/launchpad/pkg/product"
	"github.com/Mirantis/launchpad/pkg/product/mke"
	"github.com/Mirantis/launchpad/pkg/util/redact"
	"github.com/a8m/envsubst"
	"github.com/k0sproject/rig/v2/cmd"
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

	// noUnset: fail loudly instead of silently substituting an empty string
	// when a literal, unescaped "$word" in the config (e.g. a generated
	// password) happens to look like an environment variable reference but
	// isn't one -- prior to this, such values were silently truncated at
	// the "$", producing a corrupted credential with no indication of why.
	// noDigit: "$" followed by a digit (e.g. a password containing "$5986")
	// can never be a valid environment variable name, so leave it as
	// literal text rather than treating it as a reference.
	// A literal "$" can still be produced by escaping it as "$$".
	plain, err := envsubst.BytesRestrictedNoDigit(data, true, false, true)
	if err != nil {
		return nil, fmt.Errorf("failed to substitute environment variables: %w", err)
	}

	cfg := string(plain)
	if !cmd.DisableRedact {
		cfg = redactConfigDump(plain)
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

// legacyRedactRegexp is the pre-PRODENG-3922 masking, kept only as a
// fallback for configurations that cannot be parsed as YAML.
//
// It is case sensitive, so it never matched the schema's camelCase
// adminPassword or adminUsername, and it does not cover keyData, certData,
// configData or host environment variables. That is why it was replaced.
var legacyRedactRegexp = regexp.MustCompile(`(username|password)([:= ]) ?\S+`)

// redactConfigDump masks credentials in the configuration before it is
// written to the debug log, which persists to disk.
//
// A configuration that fails to parse still gets dumped, because a debug
// dump of a broken configuration is exactly when the output is wanted. It
// gets the old regex instead, and the log says so: that masking is
// best-effort on text nothing could parse, it is not the key-based
// redaction the parsed path gets, and an operator deciding whether to paste
// the output into a ticket needs to know which one they are looking at.
func redactConfigDump(plain []byte) string {
	masked, err := redact.YAML(plain)
	if err == nil {
		return string(masked)
	}

	log.Warnf("configuration could not be parsed for redaction (%s); the dump below is masked on a best-effort basis only and may still contain credentials", err)

	return legacyRedactRegexp.ReplaceAllString(string(plain), "$1$2[REDACTED]")
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
