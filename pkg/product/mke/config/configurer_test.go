package config

import (
	"reflect"
	"testing"

	"github.com/Mirantis/launchpad/pkg/configurer/centos"
	"github.com/Mirantis/launchpad/pkg/configurer/enterpriselinux"
	"github.com/Mirantis/launchpad/pkg/configurer/oracle"
	"github.com/Mirantis/launchpad/pkg/configurer/sles"
	"github.com/Mirantis/launchpad/pkg/configurer/ubuntu"
	"github.com/Mirantis/launchpad/pkg/configurer/windows"
)

func castConfigurer(cfg interface{}) bool {
	_, ok := cfg.(HostConfigurer)
	return ok
}

// missingHostConfigurerMethods returns the names of HostConfigurer interface methods
// that the given value does not implement. Returns nil if it fully implements the interface.
func missingHostConfigurerMethods(v interface{}) []string {
	it := reflect.TypeOf((*HostConfigurer)(nil)).Elem()
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return []string{"not a struct"}
	}
	var missing []string
	for i := 0; i < it.NumMethod(); i++ {
		m := it.Method(i)
		_, ok := t.MethodByName(m.Name)
		if !ok {
			missing = append(missing, m.Name)
		}
	}
	return missing
}

func TestHostConfigurerInterface(t *testing.T) {
	configurers := []struct {
		name string
		cfg  interface{}
	}{
		{"centos.Configurer", centos.Configurer{}},
		{"enterpriselinux.Configurer", enterpriselinux.Configurer{}},
		{"enterpriselinux.Rhel", enterpriselinux.Rhel{}},
		{"oracle.Configurer", oracle.Configurer{}},
		{"sles.Configurer", sles.Configurer{}},
		{"windows.Windows2019Configurer", &windows.Windows2019Configurer{}},
		{"windows.Windows2022Configurer", &windows.Windows2022Configurer{}},
		{"windows.Windows2025Configurer", &windows.Windows2025Configurer{}},
		{"ubuntu.BionicConfigurer", ubuntu.BionicConfigurer{}},
		{"ubuntu.FocalConfigurer", ubuntu.FocalConfigurer{}},
		{"ubuntu.JammyConfigurer", ubuntu.JammyConfigurer{}},
		{"ubuntu.NobleConfigurer", ubuntu.NobleConfigurer{}},
		{"ubuntu.XenialConfigurer", ubuntu.XenialConfigurer{}},
	}
	for _, c := range configurers {
		if !castConfigurer(c.cfg) {
			missing := missingHostConfigurerMethods(c.cfg)
			t.Errorf("%s does not implement HostConfigurer; missing methods: %v", c.name, missing)
		}
	}
}

func TestHostConfigurerInterfaceMissingMethods(t *testing.T) {
	// Log which methods are missing from each configurer type (for debugging).
	configurers := []struct {
		name string
		cfg  interface{}
	}{
		{"centos.Configurer", centos.Configurer{}},
		{"enterpriselinux.Configurer", enterpriselinux.Configurer{}},
		{"enterpriselinux.Rhel", enterpriselinux.Rhel{}},
		{"oracle.Configurer", oracle.Configurer{}},
		{"sles.Configurer", sles.Configurer{}},
		{"windows.Windows2019Configurer", &windows.Windows2019Configurer{}},
		{"windows.Windows2022Configurer", &windows.Windows2022Configurer{}},
		{"windows.Windows2025Configurer", &windows.Windows2025Configurer{}},
		{"ubuntu.BionicConfigurer", ubuntu.BionicConfigurer{}},
		{"ubuntu.FocalConfigurer", ubuntu.FocalConfigurer{}},
		{"ubuntu.JammyConfigurer", ubuntu.JammyConfigurer{}},
		{"ubuntu.NobleConfigurer", ubuntu.NobleConfigurer{}},
		{"ubuntu.XenialConfigurer", ubuntu.XenialConfigurer{}},
	}
	for _, c := range configurers {
		missing := missingHostConfigurerMethods(c.cfg)
		if len(missing) > 0 {
			t.Logf("%s missing HostConfigurer methods: %v", c.name, missing)
		}
	}
}
