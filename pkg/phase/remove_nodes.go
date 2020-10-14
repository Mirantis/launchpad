package phase

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/dtr"
	"github.com/Mirantis/mcc/pkg/exec"
	"github.com/Mirantis/mcc/pkg/swarm"
	"github.com/Mirantis/mcc/pkg/ucp"
	"github.com/Mirantis/mcc/pkg/util"
	log "github.com/sirupsen/logrus"
	"gopkg.in/square/go-jose.v2/json"
)

// RemoveNodes phase implementation
type RemoveNodes struct {
	Analytics
	BasicPhase
}

type isManaged struct {
	node bool
	dtr  bool
}

type dockerContainer struct {
	ID         string `json:"Id"`
	Names      []string
	Image      string
	ImageID    string
	Command    string
	Created    int64
	Ports      []interface{}
	SizeRw     int64 `json:",omitempty"`
	SizeRootFs int64 `json:",omitempty"`
	Labels     map[string]string
	State      string
	Status     string
	HostConfig struct {
		NetworkMode string `json:",omitempty"`
	}
	NetworkSettings map[string]interface{}
	Mounts          []interface{}
}

// Title for the phase
func (p *RemoveNodes) Title() string {
	return "Remove nodes"
}

// Run removes all nodes from swarm that are labeled and not part of the current config
func (p *RemoveNodes) Run() error {
	swarmLeader := p.config.Spec.SwarmLeader()

	nodeIDs, err := p.currentNodeIDs(p.config)
	if err != nil {
		return err
	}
	swarmIDs, err := p.swarmNodeIDs(swarmLeader)
	if err != nil {
		return err
	}
	for _, nodeID := range swarmIDs {
		managed := p.isManagedByUs(swarmLeader, nodeID)
		if !util.StringSliceContains(nodeIDs, nodeID) && managed.node {
			// If the node is a managed dtr node in addition to a managed
			// launchpad node, first remove DTR
			if managed.dtr {
				// Check to see if the config contains any left over DTR nodes,
				// if it doesn't just call CleanupDtrs to remove
				dtrs := p.config.Spec.Dtrs()
				if len(dtrs) == 0 {
					// All of the DTRs were removed from config, just remove
					// them forcefully since we don't care about sustaining
					// quorum
					log.Debug("No further DTRs found in config, cleaning up DTR components")
					err := dtr.CleanupDtrs(dtrs, swarmLeader)
					if err != nil {
						return err
					}
				}
				// Get the hostname from the nodeID inspect
				hostname, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf(`node inspect %s --format {{.Description.Hostname}}`, nodeID))
				if err != nil {
					return fmt.Errorf("failed to obtain hostname of DTR managed node: %s from swarm: %s", nodeID, err)
				}
				// Using an httpClient, reach out to the UCP API to obtain the
				// full list of running containers so replicaID associated with
				// hostname can be determined
				replicaID, err := p.getReplicaIDFromHostname(p.config, swarmLeader, hostname)
				if err != nil {
					return err
				}
				log.Debugf("Obtained replicaID: %s from node intending to be removed", replicaID)

				// Use that hostname to issue a remove from bootstrap
				err = p.removeDtrNode(p.config, replicaID)
				if err != nil {
					return err
				}
			}

			err = p.removeNode(swarmLeader, nodeID)
			if err != nil {
				return err
			}
		}
	}

	return nil

}

func (p *RemoveNodes) currentNodeIDs(config *api.ClusterConfig) ([]string, error) {
	nodeIDs := []string{}
	for _, h := range config.Spec.Hosts {
		nodeID, err := swarm.NodeID(h)
		if err != nil {
			return []string{}, err
		}
		nodeIDs = append(nodeIDs, nodeID)
	}
	return nodeIDs, nil
}

func (p *RemoveNodes) swarmNodeIDs(swarmLeader *api.Host) ([]string, error) {
	output, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf(`node ls --format="{{.ID}}"`))
	if err != nil {
		log.Errorln(output)
		return []string{}, err
	}
	return strings.Split(output, "\n"), nil
}

func (p *RemoveNodes) removeNode(swarmLeader *api.Host, nodeID string) error {
	nodeAddr, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf(`node inspect %s --format {{.Status.Addr}}`, nodeID))
	if err != nil {
		return err
	}
	log.Infof("%s: removing orphan node %s", swarmLeader, nodeAddr)
	nodeRole, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf(`node inspect %s --format {{.Spec.Role}}`, nodeID))
	if err != nil {
		return err
	}
	if nodeRole == "manager" {
		log.Infof("%s: demoting orphan node %s", swarmLeader, nodeAddr)
		err = swarmLeader.Exec(swarmLeader.Configurer.DockerCommandf(`node demote %s`, nodeID))
		if err != nil {
			return err
		}
		log.Infof("%s: orphan node %s demoted", swarmLeader, nodeAddr)
	}

	log.Infof("%s: draining orphan node %s", swarmLeader, nodeAddr)
	drainCmd := swarmLeader.Configurer.DockerCommandf("node update --availability drain %s", nodeID)
	err = swarmLeader.Exec(drainCmd)
	if err != nil {
		return err
	}
	time.Sleep(30 * time.Second)
	log.Infof("%s: orphan node %s drained", swarmLeader, nodeAddr)

	removeCmd := swarmLeader.Configurer.DockerCommandf("node rm --force %s", nodeID)
	err = swarmLeader.Exec(removeCmd)
	if err != nil {
		return err
	}
	log.Infof("%s: removed orphan node %s", swarmLeader, nodeAddr)
	return nil
}

func (p *RemoveNodes) removeDtrNode(config *api.ClusterConfig, replicaID string) error {
	dtrLeader := config.Spec.DtrLeader()
	ucpFlags := dtr.BuildUcpFlags(config)

	runFlags := []string{"--rm", "-i"}
	if dtrLeader.Configurer.SELinuxEnabled() {
		runFlags = append(runFlags, "--security-opt label=disable")
	}

	removeFlags := []string{
		fmt.Sprintf("--replica-ids %s", replicaID),
		fmt.Sprintf("--existing-replica-id %s", config.Spec.Dtr.Metadata.DtrLeaderReplicaID),
	}
	removeFlags = append(removeFlags, ucpFlags...)
	for _, f := range dtr.PluckSharedInstallFlags(config.Spec.Dtr.InstallFlags, dtr.SharedInstallRemoveFlags) {
		removeFlags = append(removeFlags, f)
	}

	removeCmd := dtrLeader.Configurer.DockerCommandf("run %s %s remove %s", strings.Join(runFlags, " "), config.Spec.Dtr.GetBootstrapperImage(), strings.Join(removeFlags, " "))
	log.Debugf("%s: Removing DTR replica %s from cluster", dtrLeader, replicaID)
	err := dtrLeader.Exec(removeCmd, exec.StreamOutput())
	if err != nil {
		return NewError("Failed to run DTR remove")
	}
	return nil
}

// isManagedByUs returns a struct of isManaged which contains two bools, one
// which declares node wide management and one which declares dtr management
func (p *RemoveNodes) isManagedByUs(swarmLeader *api.Host, nodeID string) isManaged {
	labels, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf(`node inspect %s --format="{{json .Spec.Labels}}"`, nodeID))
	var managed isManaged
	if err != nil {
		return managed
	}
	managed.node = strings.Contains(labels, `"com.mirantis.launchpad.managed":"true"`)
	managed.dtr = strings.Contains(labels, `"com.mirantis.launchpad.managed.dtr":"true"`)
	return managed
}

// getReplicaIDFromHostname retreives the replicaID from the container name
// associated with hostname
func (p *RemoveNodes) getReplicaIDFromHostname(config *api.ClusterConfig, swarmLeader *api.Host, hostname string) (string, error) {
	// Setup httpClient
	tlsConfig, err := ucp.GetTLSConfigFrom(swarmLeader, config.Spec.Ucp.ImageRepo, config.Spec.Ucp.Version)
	if err != nil {
		return "", fmt.Errorf("error getting TLS config: %w", err)
	}
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
	ucpURL, err := config.Spec.UcpURL()
	if err != nil {
		return "", err
	}

	// Get a UCP token
	token, err := ucp.GetUCPToken(client, ucpURL, config.Spec.Ucp.InstallFlags.GetValue("--admin-username"), config.Spec.Ucp.InstallFlags.GetValue("--admin-password"))
	if err != nil {
		return "", fmt.Errorf("Failed to get auth token: %s", err)
	}

	// Build the query
	ucpURL.Path = "/containers/json"
	ucpURL.Query().Add("filters", fmt.Sprintf(`{"ancestor": ["dtr-nginx:%s"]}`, config.Spec.Dtr.Version))
	ucpURL.Query().Add("size", "false")
	req, err := http.NewRequest("GET", ucpURL.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("unexpected response code: %d from %s endpoint: %s", resp.StatusCode, ucpURL.String(), err)
	}

	var containersResponse []dockerContainer
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	err = json.Unmarshal(respBody, &containersResponse)
	if err != nil {
		return "", err
	}

	// Iterate the containersResponse and check for hostname in the container
	// names, even though regex is slow it's the safer choice here
	var replicaID string
	re, _ := regexp.Compile(`\s*(\d{12})`)
	for _, container := range containersResponse {
		for _, n := range container.Names {
			if strings.HasPrefix(n, fmt.Sprintf("/%s", hostname)) {
				replicaID = re.FindString(n)
				if replicaID == "" {
					return "", fmt.Errorf("retrieved blank replicaID from hostname: %s", hostname)
				}
				return replicaID, nil
			}
		}
	}
	return "", fmt.Errorf("failed to obtain replicaID from hostname: %s", hostname)
}
