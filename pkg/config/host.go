package config

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/creasty/defaults"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/crypto/ssh"

	log "github.com/sirupsen/logrus"
)

// RemoteHost interface defines the connection (ssh) related interface each remote host should implement
type RemoteHost interface {
	Connect() error
	Disconnect() error
}

// OsRelease host operating system info
type OsRelease struct {
	ID      string
	IDLike  string
	Name    string
	Version string
}

// HostMetadata resolved metadata for host
type HostMetadata struct {
	Hostname        string
	InternalAddress string
	Os              *OsRelease
}

// Hosts is the type alias for slice of Hosts
type Hosts []*Host

// Host contains all the needed details to work with hosts
type Host struct {
	Address          string   `yaml:"address" validate:"required,hostname|ip"`
	User             string   `yaml:"user" validate:"omitempty,gt=2" default:"root"`
	SSHPort          int      `yaml:"sshPort" default:"22" validate:"gt=0,lte=65535"`
	SSHKeyPath       string   `yaml:"sshKeyPath" validate:"file" default:"~/.ssh/id_rsa"`
	Role             string   `yaml:"role" validate:"oneof=controller worker"`
	ExtraArgs        []string `yaml:"extraArgs"`
	PrivateInterface string   `yaml:"privateInterface" default:"eth0" validate:"gt=2"`
	Metadata         *HostMetadata
	Configurer       HostConfigurer

	sshClient *ssh.Client
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (h *Host) UnmarshalYAML(unmarshal func(interface{}) error) error {
	defaults.Set(h)
	h.SSHKeyPath, _ = homedir.Expand(h.SSHKeyPath)

	type plain Host
	if err := unmarshal((*plain)(h)); err != nil {
		return err
	}

	return nil
}

// Connect to the host
func (h *Host) Connect() error {
	key, err := ioutil.ReadFile(h.SSHKeyPath)
	if err != nil {
		return err
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return err
	}
	config := ssh.ClientConfig{
		User: h.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	address := fmt.Sprintf("%s:%d", h.Address, h.SSHPort)

	client, err := ssh.Dial("tcp", address, &config)
	if err != nil {
		return err
	}
	h.sshClient = client

	return nil
}

// Exec a command on the host and streams the logs
func (h *Host) Exec(cmd string) error {
	session, err := h.sshClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	modes := ssh.TerminalModes{}
	err = session.RequestPty("xterm", 80, 40, modes)
	if err != nil {
		return err
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return err
	}

	log.Debugf("executing command: %s", cmd)
	if err := session.Start(cmd); err != nil {
		return err
	}

	multiReader := io.MultiReader(stdout, stderr)
	outputScanner := bufio.NewScanner(multiReader)

	for outputScanner.Scan() {
		log.Debugf("%s:  %s", h.Address, outputScanner.Text())
	}
	if err := outputScanner.Err(); err != nil {
		log.Errorf("%s:  %s", h.Address, err.Error())
	}

	return session.Wait()
}

// ExecWithOutput execs a command on the host and return output
func (h *Host) ExecWithOutput(cmd string) (string, error) {
	session, err := h.sshClient.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return trimOutput(output), err
	}

	return trimOutput(output), nil
}

func trimOutput(output []byte) string {
	if len(output) > 0 {
		return strings.TrimSpace(string(output))
	}

	return ""
}

// PullImage pulls the named docker image on the host
func (h *Host) PullImage(name string) error {
	output, err := h.ExecWithOutput(h.Configurer.DockerCommandf("pull %s", name))
	if err != nil {
		log.Warnf("%s: failed to pull image %s: \n%s", h.Address, name, output)
		return err
	}
	return nil
}

// SwarmAddress determines the swarm address for the host
func (h *Host) SwarmAddress() string {
	return fmt.Sprintf("%s:%d", h.Metadata.InternalAddress, 2377)
}
