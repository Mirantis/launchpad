package api

import (
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Mirantis/mcc/pkg/constant"
)

func TestMSR2Config_UseLegacyImageRepo(t *testing.T) {
	cfg := MSR2Config{}
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

func TestMSR3Config_ConfigureCRD(t *testing.T) {
	cfg := MSR3Config{
		Name:         "SoMeNAME",
		Version:      "1.2.3",
		ImageRepo:    "registry.example.com/super/cool/repo",
		ReplicaCount: 3,
	}
	err := cfg.configureCRD()
	require.NoError(t, err)

	actualName, found, err := unstructured.NestedString(cfg.CRD.Object, "metadata", "name")
	require.True(t, found)
	require.NoError(t, err)

	assert.Equal(t, "somename", actualName)

	imageMap, found, err := unstructured.NestedStringMap(cfg.CRD.Object, "spec", "image")
	require.True(t, found)
	require.NoError(t, err)

	for _, expectedKey := range []string{
		"registry", "repository", "tag",
	} {
		_, found := imageMap[expectedKey]
		require.True(t, found, "expected key %q in image map", expectedKey)
	}

	assert.Equal(t, "registry.example.com", imageMap["registry"])
	assert.Equal(t, "super/cool/repo", imageMap["repository"])
	assert.Equal(t, "1.2.3", imageMap["tag"])

	validateReplicaCountExists(t, cfg.CRD.Object, 3)

	// Since 1 is the default replica count, it should not be set in the CRD.
	cfg.ReplicaCount = 1
	err = cfg.configureCRD()
	require.NoError(t, err)

	validateNoReplicaCountFields(t, cfg.CRD.Object)
}

func validateReplicaCountExists(t *testing.T, obj map[string]interface{}, expected int64) {
	t.Helper()

	for _, fields := range [][]string{
		{"spec", "nginx", "replicaCount"},
		{"spec", "garant", "replicaCount"},
		{"spec", "api", "replicaCount"},
		{"spec", "notarySigner", "replicaCount"},
		{"spec", "notaryServer", "replicaCount"},
		{"spec", "registry", "replicaCount"},
		{"spec", "rethinkdb", "cluster", "replicaCount"},
		{"spec", "rethinkdb", "proxy", "replicaCount"},
		{"spec", "enzi", "api", "replicaCount"},
		{"spec", "enzi", "worker", "replicaCount"},
	} {
		actualReplicaCount, found, err := unstructured.NestedInt64(obj, fields...)
		require.True(t, found)
		require.NoError(t, err)

		assert.Equal(t, expected, actualReplicaCount, "%s should be %d", strings.Join(fields, "."), expected)
	}

	actualPreset, found, err := unstructured.NestedString(obj, "spec", "podAntiAffinityPreset")
	require.True(t, found)
	require.NoError(t, err)

	assert.Equal(t, "hard", actualPreset)
}

func validateNoReplicaCountFields(t *testing.T, obj map[string]interface{}) {
	t.Helper()

	for _, fields := range [][]string{
		{"spec", "nginx", "replicaCount"},
		{"spec", "garant", "replicaCount"},
		{"spec", "api", "replicaCount"},
		{"spec", "notarySigner", "replicaCount"},
		{"spec", "notaryServer", "replicaCount"},
		{"spec", "registry", "replicaCount"},
		{"spec", "rethinkdb", "cluster", "replicaCount"},
		{"spec", "rethinkdb", "proxy", "replicaCount"},
		{"spec", "enzi", "api", "replicaCount"},
		{"spec", "enzi", "worker", "replicaCount"},
	} {
		_, found, err := unstructured.NestedInt64(obj, fields...)
		assert.False(t, found)
		assert.NoError(t, err)
	}
}
