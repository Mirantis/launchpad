package v1beta1

import (
	"github.com/Mirantis/mcc/pkg/connection/ssh"
	"github.com/creasty/defaults"
	"github.com/mitchellh/go-homedir"
)

// Host contains all the needed details to work with hosts
type SSH struct {
	User    string `yaml:"user" validate:"omitempty,gt=2" default:"root"`
	Port    int    `yaml:"port" default:"22" validate:"gt=0,lte=65535"`
	KeyPath string `yaml:"keyPath" validate:"omitempty,file" default:"~/.ssh/id_rsa"`
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

func (s *SSH) NewConnection(address string) *ssh.Connection {
	return &ssh.Connection{
		Address: address,
		User:    s.User,
		Port:    s.Port,
		KeyPath: s.KeyPath,
	}
}

func DefaultSSH() *SSH {
	ssh := SSH{}
	defaults.Set(&ssh)
	return &ssh
}
