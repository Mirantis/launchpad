package api

import (
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/Mirantis/mcc/pkg/constant"
)

func TestMSR2Config_UseLegacyImageRepo(t *testing.T) {
	cfg := MSR2Config{}
	// >=3.1.15 || >=3.2.8 || >=3.3.2 is "mirantis"
	legacyVersions := []string{
		"2.8.1",
		"2.7.7",
		"2.6.14",
		"2.6.14-rc1",
		"2.5.2",
		"1.2.3",
	}
	modernVersions := []string{
		"2.8.2",
		"2.9.3",
		"2.7.8",
		"2.6.15",
		"2.6.15-rc5",
		"4.0.0",
	}

	for _, vs := range legacyVersions {
		v, _ := version.NewVersion(vs)
		require.True(t, cfg.UseLegacyImageRepo(v), "should be true for %s", vs)
	}

	for _, vs := range modernVersions {
		v, _ := version.NewVersion(vs)
		require.False(t, cfg.UseLegacyImageRepo(v), "should be false for %s", vs)
	}
}

func TestMSR2Config_LegacyDefaultVersionRepo(t *testing.T) {
	cfg := MSR2Config{}
	err := yaml.Unmarshal([]byte("version: 2.8.1"), &cfg)
	require.NoError(t, err)
	require.Equal(t, constant.ImageRepoLegacy, cfg.ImageRepo)
}

func TestMSR2Config_ModernDefaultVersionRepo(t *testing.T) {
	cfg := MSR2Config{}
	err := yaml.Unmarshal([]byte("version: 2.8.2"), &cfg)
	require.NoError(t, err)
	require.Equal(t, constant.ImageRepo, cfg.ImageRepo)
}

func TestMSR2Config_CustomRepo(t *testing.T) {
	cfg := MSR2Config{}
	err := yaml.Unmarshal([]byte("version: 2.8.2\nimageRepo: foo.foo/foo"), &cfg)
	require.NoError(t, err)
	require.Equal(t, "foo.foo/foo", cfg.ImageRepo)
	cfg = MSR2Config{}
	err = yaml.Unmarshal([]byte("version: 2.8.1\nimageRepo: foo.foo/foo"), &cfg)
	require.NoError(t, err)
	require.Equal(t, "foo.foo/foo", cfg.ImageRepo)
}

// TestMSR2Config_YAMLKeysDoNotOverlap tests that the yaml keys in MSR2Config and
// MSR3Config do not overlap.  This is important as the MSR2 and MSR3 configs
// are inlined under the 'msr' parent key.  During unmarshaling, the yaml
// keys should be unique to ensure that the correct version structs are
// appropriately populated.
func TestMSR2Config_YAMLKeysDoNotOverlap(t *testing.T) {
	a := extractYAMLTags(t, MSR2Config{})
	b := extractYAMLTags(t, MSR3Config{})

	for _, key := range a {
		assert.NotContainsf(t, b, key, "yaml tag: %q should not exist in both MSR2Config and MSR3Config types", key)
	}
}

// extractYAML tags iterates v's struct fields and returns a sorted slice of
// string containing yaml tags.
func extractYAMLTags(t *testing.T, v interface{}) []string {
	t.Helper()

	typ := reflect.TypeOf(v)
	if typ.Kind() != reflect.Struct {
		t.Fatalf("expected struct, got %v", typ.Kind())
	}

	// Iterate the struct fields and create a map of field names to yaml keys.
	var final []string
	for i := 0; i < typ.NumField(); i++ {
		fld := typ.Field(i)
		if keyName := fld.Tag.Get("yaml"); keyName != "" {
			if strings.Contains(keyName, ",") {
				k := strings.Split(keyName, ",")
				final = append(final, k[0])
			} else {
				final = append(final, keyName)
			}
		}
	}

	return final
}
