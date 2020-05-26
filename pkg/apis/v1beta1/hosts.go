package v1beta1

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/Mirantis/mcc/pkg/util"
	"github.com/creasty/defaults"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

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
	EngineVersion   string
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
	Role             string   `yaml:"role" validate:"oneof=manager worker"`
	ExtraArgs        []string `yaml:"extraArgs"`
	PrivateInterface string   `yaml:"privateInterface" default:"eth0" validate:"gt=2"`
	Metadata         *HostMetadata
	Configurer       HostConfigurer

	sshClient *ssh.Client
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (h *Host) UnmarshalYAML(unmarshal func(interface{}) error) error {
	defaults.Set(h)
	// Need to expand possible ~... paths so validation will pass
	h.SSHKeyPath, _ = homedir.Expand(h.SSHKeyPath)
	type plain Host
	if err := unmarshal((*plain)(h)); err != nil {
		return err
	}

	return nil
}

// Connect to the host
func (h *Host) Connect() error {
	key, err := util.LoadExternalFile(h.SSHKeyPath)
	if err != nil {
		return err
	}

	config := &ssh.ClientConfig{
		User:            h.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	address := fmt.Sprintf("%s:%d", h.Address, h.SSHPort)

	sshAgentSock := os.Getenv("SSH_AUTH_SOCK")
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil && sshAgentSock == "" {
		return err
	}
	if err == nil {
		config.Auth = append(config.Auth, ssh.PublicKeys(signer))
	}

	if sshAgentSock != "" {
		sshAgent, err := net.Dial("unix", sshAgentSock)
		if err != nil {
			return fmt.Errorf("cannot connect to SSH agent auth socket %s: %s", sshAgentSock, err)
		}
		log.Debugf("using SSH auth sock %s", sshAgentSock)
		config.Auth = append(config.Auth, ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers))
	}

	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return err
	}
	h.sshClient = client

	return nil
}

// ExecCmd a command on the host piping stdin and streams the logs
func (h *Host) ExecCmd(cmd string, stdin string, streamStdout bool, sensitiveCommand bool) error {
	session, err := h.sshClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	if stdin == "" && !h.IsWindows() {
		// FIXME not requesting a pty for commands with stdin input for now,
		// as it appears the pipe doesn't get closed with stdinpipe.Close()
		modes := ssh.TerminalModes{}
		err = session.RequestPty("xterm", 80, 40, modes)
		if err != nil {
			return err
		}
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return err
	}

	stdinPipe, err := session.StdinPipe()
	if err != nil {
		return err
	}

	if !sensitiveCommand {
		log.Debugf("%s: executing command: %s", h.Address, cmd)
	}

	if err := session.Start(cmd); err != nil {
		return err
	}

	if stdin != "" {
		log.Debugf("%s: writing data to command stdin: %s", h.Address, stdin)
		go func() {
			defer stdinPipe.Close()
			io.WriteString(stdinPipe, stdin)
		}()
	}

	multiReader := io.MultiReader(stdout, stderr)
	outputScanner := bufio.NewScanner(multiReader)

	for outputScanner.Scan() {
		if streamStdout {
			log.Infof("%s:  %s", h.Address, outputScanner.Text())
		} else {
			log.Debugf("%s:  %s", h.Address, outputScanner.Text())
		}
	}
	if err := outputScanner.Err(); err != nil {
		log.Errorf("%s:  %s", h.Address, err.Error())
	}

	return session.Wait()
}

// Exec a command on the host and streams the logs
func (h *Host) Exec(cmd string) error {
	return h.ExecCmd(cmd, "", false, false)
}

// Execf a printf-formatted command on the host and streams the logs
func (h *Host) Execf(cmd string, args ...interface{}) error {
	return h.Exec(fmt.Sprintf(cmd, args...))
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

// WriteFile writes file to host with given contents
func (h *Host) WriteFile(path string, data string, permissions string) error {
	tempFile, _ := h.ExecWithOutput("mktemp")
	err := h.ExecCmd(fmt.Sprintf("cat > %s && (sudo install -m %s %s %s || (rm %s; exit 1))", tempFile, permissions, tempFile, path, tempFile), data, false, true)
	if err != nil {
		return err
	}
	return nil
}

func trimOutput(output []byte) string {
	if len(output) > 0 {
		return strings.TrimSpace(string(output))
	}

	return ""
}

// AuthenticateDocker performs a docker login on the host using local REGISTRY_USERNAME
// and REGISTRY_PASSWORD when set
func (h *Host) AuthenticateDocker(server string) error {
	if user := os.Getenv("REGISTRY_USERNAME"); user != "" {
		pass := os.Getenv("REGISTRY_PASSWORD")
		if pass == "" {
			return fmt.Errorf("%s: REGISTRY_PASSWORD not set", h.Address)
		}
		log.Infof("%s: authenticating docker", h.Address)
		old := log.GetLevel()
		log.SetLevel(log.ErrorLevel)
		err := h.ExecCmd(h.Configurer.DockerCommandf("login -u %s --password-stdin %s", user, server), pass, false, true)
		log.SetLevel(old)

		if err != nil {
			return fmt.Errorf("%s: failed to authenticate docker: %s", h.Address, err)
		}
	} else {
		log.Debugf("%s: REGISTRY_USERNAME not set, not authenticating", h.Address)
	}
	return nil
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

// IsWindows returns true if host has been detected running windows
func (h *Host) IsWindows() bool {
	if h.Metadata.Os == nil {
		return false
	}
	return strings.HasPrefix(h.Metadata.Os.ID, "windows-")
}
