package api

import (
	"testing"

	"github.com/Mirantis/mcc/pkg/configurer/centos"
	"github.com/Mirantis/mcc/pkg/configurer/enterpriselinux"
	"github.com/Mirantis/mcc/pkg/configurer/oracle"
	"github.com/Mirantis/mcc/pkg/configurer/sles"
	"github.com/Mirantis/mcc/pkg/configurer/ubuntu"
	"github.com/Mirantis/mcc/pkg/configurer/windows"
	"github.com/stretchr/testify/require"
)

func castConfigurer(cfg interface{}) bool {
	_, ok := cfg.(HostConfigurer)
	return ok
}

func TestHostConfigurerInterface(t *testing.T) {
	require.True(t, castConfigurer(centos.Configurer{}), "configurer does not implement HostConfigurer")
	require.True(t, castConfigurer(enterpriselinux.Configurer{}), "configurer does not implement HostConfigurer")
	require.True(t, castConfigurer(enterpriselinux.Rhel{}), "configurer does not implement HostConfigurer")
	require.True(t, castConfigurer(oracle.Configurer{}), "configurer does not implement HostConfigurer")
	require.True(t, castConfigurer(sles.Configurer{}), "configurer does not implement HostConfigurer")
	require.True(t, castConfigurer(windows.Windows2019Configurer{}), "configurer does not implement HostConfigurer")
	require.True(t, castConfigurer(windows.Windows2022Configurer{}), "configurer does not implement HostConfigurer")
	require.True(t, castConfigurer(ubuntu.BionicConfigurer{}), "configurer does not implement HostConfigurer")
	require.True(t, castConfigurer(ubuntu.FocalConfigurer{}), "configurer does not implement HostConfigurer")
	require.True(t, castConfigurer(ubuntu.JammyConfigurer{}), "configurer does not implement HostConfigurer")
	require.True(t, castConfigurer(ubuntu.XenialConfigurer{}), "configurer does not implement HostConfigurer")
}
