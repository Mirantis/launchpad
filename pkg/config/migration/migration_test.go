package migration

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMigrateMalformedAPIVersion covers the dispatcher's handling of an
// apiVersion that is absent or is not a string. Both previously reached an
// unguarded type assertion -- the comma-ok was bound to the map lookup, not to
// the assertion -- and panicked before any structural validation ran.
func TestMigrateMalformedAPIVersion(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]interface{}
		wantErr bool
	}{
		{
			name:    "missing apiVersion",
			data:    map[string]interface{}{"kind": "mke"},
			wantErr: true,
		},
		{
			name:    "nil apiVersion",
			data:    map[string]interface{}{"apiVersion": nil},
			wantErr: true,
		},
		{
			name:    "integer apiVersion",
			data:    map[string]interface{}{"apiVersion": 3},
			wantErr: true,
		},
		{
			name:    "boolean apiVersion",
			data:    map[string]interface{}{"apiVersion": true},
			wantErr: true,
		},
		{
			name:    "mapping apiVersion",
			data:    map[string]interface{}{"apiVersion": map[string]interface{}{"nested": "value"}},
			wantErr: true,
		},
		{
			name:    "list apiVersion",
			data:    map[string]interface{}{"apiVersion": []interface{}{"a", "b"}},
			wantErr: true,
		},
		{
			name:    "current apiVersion has no migrator and is not an error",
			data:    map[string]interface{}{"apiVersion": "launchpad.mirantis.com/mke/v1.6"},
			wantErr: false,
		},
		{
			name:    "unknown string apiVersion has no migrator and is not an error",
			data:    map[string]interface{}{"apiVersion": "launchpad.mirantis.com/not-a-real-version"},
			wantErr: false,
		},
		{
			name:    "empty string apiVersion has no migrator and is not an error",
			data:    map[string]interface{}{"apiVersion": ""},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// require.NotPanics is the point of the test: every case below
			// reached a panic before the assertion was guarded.
			var err error
			require.NotPanics(t, func() { err = Migrate(tt.data) })

			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, ErrInvalidAPIVersion)
		})
	}
}

func TestMigrateRunsRegisteredMigrators(t *testing.T) {
	// A migrator that advances the apiVersion must still be reached, so the
	// guard cannot have broken the normal chain.
	const from = "test.launchpad/v1"
	const to = "test.launchpad/v2"

	called := 0
	Register(from, func(data map[string]interface{}) error {
		called++
		data["apiVersion"] = to
		return nil
	})
	t.Cleanup(func() { delete(migrators, from) })

	data := map[string]interface{}{"apiVersion": from}
	require.NoError(t, Migrate(data))
	require.Equal(t, 1, called)
	require.Equal(t, to, data["apiVersion"])
}
