package v1beta2

import (
	"github.com/Mirantis/mcc/pkg/connection/winrm"
	"github.com/creasty/defaults"
	"github.com/mitchellh/go-homedir"
)

// WinRM contains configuration options for a WinRM connection
type WinRM struct {
	User          string `yaml:"user" validate:"omitempty,gt=2" default:"Administrator"`
	Port          int    `yaml:"port" default:"5985" validate:"gt=0,lte=65535"`
	Password      string `yaml:"password,omitempty"`
	UseHTTPS      bool   `yaml:"useHTTPS" default:"false"`
	Insecure      bool   `yaml:"insecure" default:"false"`
	UseNTLM       bool   `yaml:"useNTLM" default:"false"`
	CACertPath    string `yaml:"caCertPath,omitempty" validate:"omitempty,file"`
	CertPath      string `yaml:"certPath,omitempty" validate:"omitempty,file"`
	KeyPath       string `yaml:"keyPath,omitempty" validate:"omitempty,file"`
	TLSServerName string `yaml:"tlsServerName,omitempty"`
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (w *WinRM) UnmarshalYAML(unmarshal func(interface{}) error) error {
	defaults.Set(w)

	type yWinRM WinRM
	yw := (*yWinRM)(w)

	if err := unmarshal(yw); err != nil {
		return err
	}

	w.CACertPath, _ = homedir.Expand(w.CACertPath)
	w.CertPath, _ = homedir.Expand(w.CACertPath)
	w.KeyPath, _ = homedir.Expand(w.CACertPath)

	return nil
}

// NewConnection returns a new WinRM connection instance
func (w WinRM) NewConnection(address string) *winrm.Connection {
	return &winrm.Connection{
		Address:       address,
		User:          w.User,
		Port:          w.Port,
		Password:      w.Password,
		UseHTTPS:      w.UseHTTPS,
		Insecure:      w.Insecure,
		UseNTLM:       w.UseNTLM,
		CACertPath:    w.CACertPath,
		CertPath:      w.CertPath,
		KeyPath:       w.KeyPath,
		TLSServerName: w.TLSServerName,
	}
}
