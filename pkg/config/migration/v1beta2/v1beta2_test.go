package v1beta2

import (
	"testing"

	"github.com/Mirantis/launchpad/pkg/config/migration"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestMigration(t *testing.T) {
	b2 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1beta2"
kind: UCP
spec:
  engine:
    installURL: http://get.example.com/
  hosts:
    - address: "10.0.0.1"
      role: "manager"
`)
	// go's YAML marshal does not add the --- header
	b3 := []byte(`apiVersion: launchpad.mirantis.com/v1beta3
kind: DockerEnterprise
spec:
  engine:
    installURLLinux: http://get.example.com/
  hosts:
  - address: 10.0.0.1
    role: manager
`)
	in := make(map[string]interface{})
	require.NoError(t, yaml.Unmarshal(b2, in))
	require.NoError(t, Migrate(in))
	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, b3, out)
}

// TestMigrateMalformedSpec covers the unguarded type assertion on the spec
// section. plain["spec"].(map[interface{}]interface{})["engine"] panicked for
// any spec that was present but not a mapping.
func TestMigrateMalformedSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    interface{}
		wantErr bool
	}{
		{name: "string spec", spec: "not-a-mapping", wantErr: true},
		{name: "integer spec", spec: 42, wantErr: true},
		{name: "list spec", spec: []interface{}{"a"}, wantErr: true},
		{name: "string-keyed mapping", spec: map[string]interface{}{"engine": "x"}, wantErr: true},
		{name: "absent spec", spec: nil, wantErr: false},
		{name: "empty mapping", spec: map[interface{}]interface{}{}, wantErr: false},
		{
			name:    "mapping with engine",
			spec:    map[interface{}]interface{}{"engine": map[interface{}]interface{}{"installURL": "http://example.test/i.sh"}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plain := map[string]interface{}{"apiVersion": "launchpad.mirantis.com/v1beta2"}
			if tt.spec != nil {
				plain["spec"] = tt.spec
			}

			var err error
			require.NotPanics(t, func() { err = Migrate(plain) })

			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, migration.ErrInvalidSpec)
		})
	}
}

func TestMigrateRenamesInstallURL(t *testing.T) {
	engine := map[interface{}]interface{}{"installURL": "http://example.test/i.sh"}
	plain := map[string]interface{}{
		"apiVersion": "launchpad.mirantis.com/v1beta2",
		"kind":       "UCP",
		"spec":       map[interface{}]interface{}{"engine": engine},
	}

	require.NoError(t, Migrate(plain))

	require.Equal(t, "launchpad.mirantis.com/v1beta3", plain["apiVersion"])
	require.Equal(t, "DockerEnterprise", plain["kind"])
	require.Equal(t, "http://example.test/i.sh", engine["installURLLinux"])
	require.NotContains(t, engine, "installURL")
}
