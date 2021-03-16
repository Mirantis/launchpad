package phase

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Mirantis/mcc/pkg/mke"
	"github.com/Mirantis/mcc/pkg/msr"
	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	"github.com/Mirantis/mcc/pkg/swarm"
	"github.com/Mirantis/mcc/pkg/util"
	"github.com/k0sproject/rig/exec"

	log "github.com/sirupsen/logrus"
)

// RemoveNodes phase implementation
type RemoveNodes struct {
	phase.Analytics
	phase.BasicPhase
	phase.CleanupDisabling

	cleanupMSRs   []*api.Host
	msrReplicaIDs []string
	removeNodeIDs []string
}

type isManaged struct {
	node bool
	msr  bool
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

// ShouldRun is true when spec.cluster.prune is true
func (p *RemoveNodes) ShouldRun() bool {
	if !p.Config.Spec.Cluster.Prune && (len(p.cleanupMSRs) > 0 || len(p.msrReplicaIDs) > 0 || len(p.removeNodeIDs) > 0) {
		log.Warnf("There are nodes present which are not present in configuration Spec.Hosts - to remove them, set Spec.Cluster.Prune to true")
	}

	return p.Config.Spec.Cluster.Prune
}

// Prepare finds the nodes/replica ids to be removed
func (p *RemoveNodes) Prepare(config interface{}) error {
	p.Config = config.(*api.ClusterConfig)

	swarmLeader := p.Config.Spec.SwarmLeader()

	nodeIDs, err := p.currentNodeIDs(p.Config)
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
			// If the node is a managed msr node in addition to a managed
			// launchpad node, first remove MSR
			if managed.msr {
				// Check to see if the config contains any left over MSR nodes,
				// if it doesn't just call msr.Cleanup to remove
				msrs := p.Config.Spec.MSRs()
				if len(msrs) == 0 {
					// All of the MSRs were removed from config, just remove
					// them forcefully since we don't care about sustaining
					// quorum
					p.cleanupMSRs = msrs
				}
				// Get the hostname from the nodeID inspect
				hostname, err := swarmLeader.ExecOutput(swarmLeader.Configurer.Dockerf(swarmLeader, `node inspect %s --format {{.Description.Hostname}}`, nodeID))
				if err != nil {
					return fmt.Errorf("failed to obtain hostname of MSR managed node: %s from swarm: %s", nodeID, err)
				}
				// Using an httpClient, reach out to the MKE API to obtain the
				// full list of running containers so replicaID associated with
				// hostname can be determined
				replicaID, err := p.getReplicaIDFromHostname(p.Config, swarmLeader, hostname)
				if err != nil {
					return err
				}
				log.Debugf("Obtained replicaID: %s from node intending to be removed", replicaID)

				p.msrReplicaIDs = append(p.msrReplicaIDs, replicaID)
			}

			p.removeNodeIDs = append(p.removeNodeIDs, nodeID)
		}
	}
	return nil
}

// Run removes all nodes from swarm that are labeled and not part of the current config
func (p *RemoveNodes) Run() error {
	swarmLeader := p.Config.Spec.SwarmLeader()
	if len(p.cleanupMSRs) > 0 {
		err := msr.Cleanup(p.cleanupMSRs, swarmLeader)
		if err != nil {
			return err
		}
	}

	if len(p.msrReplicaIDs) > 0 {
		for _, replicaID := range p.msrReplicaIDs {
			err := p.removemsrNode(p.Config, replicaID)
			if err != nil {
				return err
			}
		}
	}
	if len(p.removeNodeIDs) > 0 {
		for _, nodeID := range p.removeNodeIDs {
			err := p.removeNode(swarmLeader, nodeID)
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

func (p *RemoveNodes) swarmNodeIDs(h *api.Host) ([]string, error) {
	output, err := h.ExecOutput(h.Configurer.Dockerf(h, `node ls --format="{{.ID}}"`))
	if err != nil {
		log.Errorln(output)
		return []string{}, err
	}
	return strings.Split(output, "\n"), nil
}

func (p *RemoveNodes) removeNode(h *api.Host, nodeID string) error {
	nodeAddr, err := h.ExecOutput(h.Configurer.Dockerf(h, `node inspect %s --format {{.Status.Addr}}`, nodeID))
	if err != nil {
		return err
	}
	log.Infof("%s: removing orphan node %s", h, nodeAddr)
	nodeRole, err := h.ExecOutput(h.Configurer.Dockerf(h, `node inspect %s --format {{.Spec.Role}}`, nodeID))
	if err != nil {
		return err
	}
	if nodeRole == "manager" {
		log.Infof("%s: demoting orphan node %s", h, nodeAddr)
		err = h.Exec(h.Configurer.Dockerf(h, `node demote %s`, nodeID))
		if err != nil {
			return err
		}
		log.Infof("%s: orphan node %s demoted", h, nodeAddr)
	}

	log.Infof("%s: draining orphan node %s", h, nodeAddr)
	drainCmd := h.Configurer.Dockerf(h, "node update --availability drain %s", nodeID)
	err = h.Exec(drainCmd)
	if err != nil {
		return err
	}
	time.Sleep(30 * time.Second)
	log.Infof("%s: orphan node %s drained", h, nodeAddr)

	removeCmd := h.Configurer.Dockerf(h, "node rm --force %s", nodeID)
	err = h.Exec(removeCmd)
	if err != nil {
		return err
	}
	log.Infof("%s: removed orphan node %s", h, nodeAddr)
	return nil
}

func (p *RemoveNodes) removemsrNode(config *api.ClusterConfig, replicaID string) error {
	msrLeader := config.Spec.MSRLeader()
	mkeFlags := msr.BuildMKEFlags(config)

	runFlags := common.Flags{"-i"}

	if !p.CleanupDisabled() {
		runFlags.Add("--rm")
	}

	if msrLeader.Configurer.SELinuxEnabled(msrLeader) {
		runFlags.Add("--security-opt label=disable")
	}

	removeFlags := common.Flags{
		fmt.Sprintf("--replica-ids %s", replicaID),
		fmt.Sprintf("--existing-replica-id %s", msrLeader.MSRMetadata.ReplicaID),
	}
	removeFlags.MergeOverwrite(mkeFlags)
	for _, f := range msr.PluckSharedInstallFlags(config.Spec.MSR.InstallFlags, msr.SharedInstallRemoveFlags) {
		removeFlags.AddOrReplace(f)
	}

	removeCmd := msrLeader.Configurer.Dockerf(msrLeader, "run %s %s remove %s", runFlags.Join(), msrLeader.MSRMetadata.InstalledBootstrapImage, removeFlags.Join())
	log.Debugf("%s: Removing MSR replica %s from cluster", msrLeader, replicaID)
	err := msrLeader.Exec(removeCmd, exec.StreamOutput())
	if err != nil {
		return fmt.Errorf("%s: failed to run MSR remove: %s", msrLeader, err.Error())
	}
	return nil
}

// isManagedByUs returns a struct of isManaged which contains two bools, one
// which declares node wide management and one which declares msr management
func (p *RemoveNodes) isManagedByUs(h *api.Host, nodeID string) isManaged {
	labels, err := h.ExecOutput(h.Configurer.Dockerf(h, `node inspect %s --format="{{json .Spec.Labels}}"`, nodeID))
	var managed isManaged
	if err != nil {
		return managed
	}
	managed.node = strings.Contains(labels, `"com.mirantis.launchpad.managed":"true"`)
	managed.msr = strings.Contains(labels, `"com.mirantis.launchpad.managed.msr":"true"`)
	return managed
}

// getReplicaIDFromHostname retreives the replicaID from the container name
// associated with hostname
func (p *RemoveNodes) getReplicaIDFromHostname(config *api.ClusterConfig, h *api.Host, hostname string) (string, error) {
	// Setup httpClient
	tlsConfig, err := mke.GetTLSConfigFrom(h, config.Spec.MKE.ImageRepo, config.Spec.MKE.Version)
	if err != nil {
		return "", fmt.Errorf("error getting TLS config: %w", err)
	}
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
	mkeURL, err := config.Spec.MKEURL()
	if err != nil {
		return "", err
	}

	// Get a MKE token
	token, err := mke.GetToken(client, mkeURL, config.Spec.MKE.AdminUsername, config.Spec.MKE.AdminPassword)
	if err != nil {
		return "", fmt.Errorf("failed to get auth token: %s", err.Error())
	}

	// Build the query
	mkeURL.Path = "/containers/json"
	mkeURL.Query().Add("filters", fmt.Sprintf(`{"ancestor": ["dtr-nginx:%s"]}`, config.Spec.MSR.Version))
	mkeURL.Query().Add("size", "false")
	req, err := http.NewRequest("GET", mkeURL.String(), nil)
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
		return "", fmt.Errorf("unexpected response code: %d from %s endpoint: %s", resp.StatusCode, mkeURL.String(), err)
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
