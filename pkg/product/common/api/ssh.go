package api

import (
	"github.com/creasty/defaults"
	"github.com/k0sproject/rig/connection/ssh"
	"github.com/mitchellh/go-homedir"
)

// SSH contains ssh connection configuration options
type SSH struct {
	User    string `yaml:"user" validate:"omitempty,gt=2" default:"root"`
	Port    int    `yaml:"port" default:"22" validate:"gt=0,lte=65535"`
	KeyPath string `yaml:"keyPath" validate:"omitempty,file" default:"~/.ssh/id_rsa"`
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (s *SSH) UnmarshalYAML(unmarshal func(interface{}) error) error {
	defaults.Set(s)

	type ssh SSH
	ys := (*ssh)(s)

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
	ssh.KeyPath, _ = homedir.Expand(ssh.KeyPath)
	return &ssh
}
