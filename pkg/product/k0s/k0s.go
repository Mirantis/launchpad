package k0s

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/Mirantis/mcc/pkg/exec"
	common "github.com/Mirantis/mcc/pkg/product/common/phase"

	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/k0s/api"
	k0s "github.com/Mirantis/mcc/pkg/product/k0s/phase"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// K0s is the product
type K0s struct {
	ClusterConfig api.ClusterConfig
	SkipCleanup   bool
	Debug         bool
}

// ClusterName returns the cluster name
func (p *K0s) ClusterName() string {
	return p.ClusterConfig.Metadata.Name
}

// New returns a new instance of the Docker Enterprise product
func New(data []byte) (*K0s, error) {
	c := api.ClusterConfig{}
	if err := yaml.UnmarshalStrict(data, &c); err != nil {
		return nil, err
	}

	// if err := c.Validate(); err != nil {
	// 	fmt.Print("HERE k0s")
	// 	return nil, err
	// }
	return &K0s{ClusterConfig: c}, nil
}

// Init returns an example configuration
func Init(kind string) *api.ClusterConfig {
	return api.Init(kind)
}

//Apply installs k0s on the desired host
func (p *K0s) Apply(disableCleanup, force bool) error {
	phaseManager := phase.NewManager(&p.ClusterConfig)
	phaseManager.SkipCleanup = disableCleanup
	phaseManager.IgnoreErrors = true

	phaseManager.AddPhases(
		&common.Connect{},
		&k0s.GatherFacts{},
		&k0s.PrepareConfig{},
		&k0s.InstallK0s{},
		&k0s.StartK0s{},
		&common.Disconnect{},
	)

	if err := phaseManager.Run(); err != nil {
		return err
	}
	return nil
}

//ClientConfig ...
func (p *K0s) ClientConfig() error {
	return nil
}

//Describe dumps information about the hosts
func (p *K0s) Describe(reportName string) error {

	log.Debugf("loaded cluster cfg: %+v", p.ClusterConfig)
	// cleaned := gh.CleanUpGenericMap()

	if reportName == "config" {
		encoder := yaml.NewEncoder(os.Stdout)
		return encoder.Encode(p.ClusterConfig)
	}

	phaseManager := phase.NewManager(&p.ClusterConfig)
	phaseManager.IgnoreErrors = true

	phaseManager.AddPhases(
		&common.Connect{},
		&k0s.GatherFacts{},
		&common.Disconnect{},
		&k0s.Describe{},
	)
	return phaseManager.Run()
}

//Exec ...
func (p *K0s) Exec(target string, interactive, first bool, role, cmd string) error {
	var host *api.Host
	if target == "" {
		if role != "" {
			host = p.ClusterConfig.Spec.Hosts.Find(func(h *api.Host) bool { return h.Role == role })
			if host == nil {
				return fmt.Errorf("failed to get the first host with role '%s' from configuration", role)
			}
		} else if first {
			host = p.ClusterConfig.Spec.Hosts.First()
			if host == nil {
				return fmt.Errorf("failed to get the first host from configuration")
			}
		} else {
			return fmt.Errorf("--target, --first or --role required") // feels like this is in the wrong place
		}
	} else if target == "localhost" {
		host = p.ClusterConfig.Spec.Hosts.Find(func(h *api.Host) bool {
			return h.Localhost
		})
	} else if strings.Contains(target, ":") {
		parts := strings.SplitN(target, ":", 2)
		addr := parts[0]
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			return fmt.Errorf("invalid port: %s", parts[1])
		}

		host = p.ClusterConfig.Spec.Hosts.Find(func(h *api.Host) bool {
			if h.Address != addr || h.Localhost {
				return false
			}
			if h.WinRM != nil {
				return h.WinRM.Port == port
			}
			return h.SSH.Port == port
		})
	} else {
		host = p.ClusterConfig.Spec.Hosts.Find(func(h *api.Host) bool {
			return h.Address == target
		})
	}
	if host == nil {
		return fmt.Errorf("Host with target %s not found in configuration", target)
	}

	var stdin string

	stat, err := os.Stdin.Stat()
	if err != nil {
		return err
	}

	if (stat.Mode() & os.ModeCharDevice) == 0 {
		if interactive {
			return fmt.Errorf("--interactive given but there's piped data in stdin")
		}
		data, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		stdin = string(data)
	}

	if err := host.Connect(); err != nil {
		return fmt.Errorf("Failed to connect: %s", err.Error())
	}

	log.Debugf("%s: connected", host)

	if cmd == "" {
		if stdin != "" {
			return fmt.Errorf("can't pipe to a remote shell without a command")
		}
		log.Tracef("assuming intention to run a shell with --interactive")
		return host.Connection.ExecInteractive("")
	}

	if interactive {
		log.Tracef("running interactive with cmd: %q", cmd)
		return host.Connection.ExecInteractive(cmd)
	}

	log.Tracef("running non-interactive with cmd: %q", cmd)
	return host.Exec(cmd, exec.Stdin(stdin), exec.StreamOutput())
}

//Reset ...
func (p *K0s) Reset() error {
	return fmt.Errorf("not supported")
}
