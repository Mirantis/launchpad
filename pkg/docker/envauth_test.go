package docker_test

import (
	"os"
	"testing"

	"github.com/Mirantis/launchpad/pkg/docker"
	"github.com/stretchr/testify/require"
)

// Be careful here as envs set in one test may still be set in the next test

func Test_EnvAuthTesting_Simple(t *testing.T) {
	os.Setenv("REGISTRY_USERNAME", "myuser")
	os.Setenv("REGISTRY_PASSWORD", "mypass")

	u, p, err := docker.DiscoverEnvLogin([]string{"REGISTRY_"})

	require.Equal(t, "myuser", u, "wrong username discovered")
	require.Equal(t, "mypass", p, "wrong password discovered")
	require.NoError(t, err, "unexpected error returned")
}

func Test_EnvAuthTesting_MissingPassword(t *testing.T) {
	os.Setenv("MISSINGPASSWORD_USERNAME", "myuser")

	_, _, err := docker.DiscoverEnvLogin([]string{"MISSINGPASSWORD_"})
	require.Error(t, err, "expected error not returned for missing password")
	require.ErrorIs(t, err, docker.ErrMissingPassword, "wrong error returned for missing pass envs")
}

func Test_EnvAuthTesting_MissingEnv(t *testing.T) {
	_, _, err := docker.DiscoverEnvLogin([]string{"NOTSET_"})
	require.Error(t, err, "expected error not returned for missing password")
	require.ErrorIs(t, err, docker.ErrNoEnvPasswordsFound, "wrong error returned for missing envs")
}

func Test_EnvAuthTesting_Multiple(t *testing.T) {
	os.Setenv("MULTIPLE_ONE_USERNAME", "userone")
	os.Setenv("MULTIPLE_ONE_PASSWORD", "passone")
	os.Setenv("MULTIPLE_TWO_USERNAME", "usertwo")
	os.Setenv("MULTIPLE_TWO_PASSWORD", "passtwo")

	u1, p1, err1 := docker.DiscoverEnvLogin([]string{"MULTIPLE_ONE_", "MULTIPLE_TWO_"})
	require.Equal(t, "userone", u1, "wrong username discovered")
	require.Equal(t, "passone", p1, "wrong password discovered")
	require.NoError(t, err1, "unexpected error returned")

	u2, p2, err2 := docker.DiscoverEnvLogin([]string{"DOESNOTEXIST_", "MULTIPLE_ONE_"})
	require.Equal(t, "userone", u2, "wrong username discovered")
	require.Equal(t, "passone", p2, "wrong password discovered")
	require.NoError(t, err2, "unexpected error returned")

	_, _, err := docker.DiscoverEnvLogin([]string{"NOTSET1_", "NOTSET2_"})
	require.Error(t, err, "expected error not returned for missing password")
	require.ErrorIs(t, docker.ErrNoEnvPasswordsFound, err, "wrong error returned for missing envs")
}
