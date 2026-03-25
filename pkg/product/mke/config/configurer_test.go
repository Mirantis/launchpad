package config

import (
	"github.com/Mirantis/launchpad/pkg/configurer/centos"
	"github.com/Mirantis/launchpad/pkg/configurer/enterpriselinux"
	"github.com/Mirantis/launchpad/pkg/configurer/oracle"
	"github.com/Mirantis/launchpad/pkg/configurer/sles"
	"github.com/Mirantis/launchpad/pkg/configurer/ubuntu"
	"github.com/Mirantis/launchpad/pkg/configurer/windows"
)

// Compile-time assertions that each OS configurer implements HostConfigurer.
var (
	_ HostConfigurer = centos.Configurer{}
	_ HostConfigurer = enterpriselinux.Configurer{}
	_ HostConfigurer = enterpriselinux.Rhel{}
	_ HostConfigurer = oracle.Configurer{}
	_ HostConfigurer = sles.Configurer{}
	_ HostConfigurer = windows.Windows2019Configurer{}
	_ HostConfigurer = windows.Windows2022Configurer{}
	_ HostConfigurer = windows.Windows2025Configurer{}
	_ HostConfigurer = ubuntu.BionicConfigurer{}
	_ HostConfigurer = ubuntu.FocalConfigurer{}
	_ HostConfigurer = ubuntu.JammyConfigurer{}
	_ HostConfigurer = ubuntu.NobleConfigurer{}
	_ HostConfigurer = ubuntu.XenialConfigurer{}
)
