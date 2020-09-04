package api

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
	TLSServerName string `yaml:"tlsServerName,omitempty" validate:"omitempty,hostname|ip"`
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (w *WinRM) UnmarshalYAML(unmarshal func(interface{}) error) error {
	defaults.Set(w)

	type yWinRM WinRM
	yw := (*yWinRM)(w)

	if err := unmarshal(yw); err != nil {
		return err
	}

	if len(w.CACertPath) > 0 {
		w.CACertPath, _ = homedir.Expand(w.CACertPath)
	}

	if len(w.CertPath) > 0 {
		w.CertPath, _ = homedir.Expand(w.CertPath)
	}

	if len(w.KeyPath) > 0 {
		w.KeyPath, _ = homedir.Expand(w.KeyPath)
	}

	if w.UseHTTPS && w.Port == 5985 {
		// Questionable - the user could be forcing port 5985 for HTTPS, which is now impossible.
		// (the default port for WinRM HTTP is 5985 and the default for WinRM HTTPS is 5986.)
		w.Port = 5986
	}

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
