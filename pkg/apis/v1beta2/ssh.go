package v1beta2

import (
	"github.com/Mirantis/mcc/pkg/connection/ssh"
	"github.com/creasty/defaults"
	"github.com/mitchellh/go-homedir"
)

// SSH contains ssh connection configuration options
type SSH struct {
	User    string `yaml:"user" default:"root" validate:"gt=2"`
	Port    int    `yaml:"port" default:"22" validate:"gt=0,lte=65535"`
	KeyPath string `yaml:"keyPath" default:"~/.ssh/id_rsa" validate:"file"`
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (s *SSH) UnmarshalYAML(unmarshal func(interface{}) error) error {
	defaults.Set(s)

	type ySSH SSH
	ys := (*ySSH)(s)

	if err := unmarshal(ys); err != nil {
		return err
	}

	s.KeyPath, _ = homedir.Expand(s.KeyPath)

	return nil
}

// NewConnection returns a new ssh connection instance
func (s *SSH) NewConnection(address string) *ssh.Connection {
	return &ssh.Connection{
		Address: address,
		User:    s.User,
		Port:    s.Port,
		KeyPath: s.KeyPath,
	}
}

// DefaultSSH provides an instance of ssh configuration with the defaults set
func DefaultSSH() *SSH {
	ssh := SSH{}
	defaults.Set(&ssh)
	return &ssh
}
