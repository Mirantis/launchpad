package msr2

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"

	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	"github.com/Mirantis/mcc/pkg/util/stringutil"
)

// CollectFacts gathers the current status of the installed MSR setup.
func CollectFacts(h *api.Host) (*api.MSRMetadata, error) {
	rethinkdbContainerID, err := h.ExecOutput(h.Configurer.DockerCommandf(`ps -aq --filter name=dtr-rethinkdb`))
	if err != nil {
		return nil, fmt.Errorf("failed to get MSR container ID: %w", err)
	}
	if rethinkdbContainerID == "" {
		return &api.MSRMetadata{Installed: false}, nil
	}

	version, err := h.ExecOutput(h.Configurer.DockerCommandf(`inspect %s --format '{{ index .Config.Labels "com.docker.dtr.version"}}'`, rethinkdbContainerID))
	if err != nil {
		return nil, fmt.Errorf("failed to get MSR version: %w", err)
	}
	replicaID, err := h.ExecOutput(h.Configurer.DockerCommandf(`inspect %s --format '{{ index .Config.Labels "com.docker.dtr.replica"}}'`, rethinkdbContainerID))
	if err != nil {
		return nil, fmt.Errorf("failed to get MSR replicaID: %w", err)
	}
	if version == "" || replicaID == "" {
		// If we failed to obtain either label then this MSR version does not
		// support the version Config.Labels or something else may have gone
		// wrong, attempt to pull these details with the old method
		output, err := h.ExecOutput(h.Configurer.DockerCommandf(`inspect %s --format '{{ index .Config.Labels "com.docker.compose.project"}}'`, rethinkdbContainerID))
		if err != nil {
			return nil, fmt.Errorf("failed to get MSR rethink container labels: %w", err)
		}
		outputFields := strings.Fields(output)
		if version == "" {
			version = outputFields[3]
		}
		if replicaID == "" {
			replicaID = strings.Trim(outputFields[len(outputFields)-1], ")")
		}
	}

	var bootstrapimage string
	imagename, err := h.ExecOutput(h.Configurer.DockerCommandf(`inspect %s --format '{{ .Config.Image }}'`, rethinkdbContainerID))
	if err == nil {
		repo := imagename[:strings.LastIndexByte(imagename, '/')]
		bootstrapimage = fmt.Sprintf("%s/dtr:%s", repo, version)
	}

	msrMeta := &api.MSRMetadata{
		Installed:        true,
		InstalledVersion: version,
		MSR2: api.MSR2Metadata{
			InstalledBootstrapImage: bootstrapimage,
			ReplicaID:               replicaID,
		},
	}

	return msrMeta, nil
}

// PluckSharedInstallFlags plucks the shared flag values between install and
// shared and returns a slice of flags and their values
// FIXME(squizzi): There's probably a better way to do this, this is a bit
// overkill.
func PluckSharedInstallFlags(installFlags []string, sharedFlags []string) []string {
	// Make a new map based on the given install flags and their values
	installFlagsMap := make(map[string]string, len(installFlags))
	for _, f := range installFlags {
		// Fill the new map with flag name -> flag value
		values := strings.Fields(f)
		valueLen := len(values)
		if valueLen == 1 {
			// If the flag is a bool flag, drop the value
			installFlagsMap[values[0]] = ""
		}
		if valueLen >= 2 {
			// If the flag has one or more values, assign those values
			// as string in the map
			installFlagsMap[values[0]] = strings.Join(values[1:], " ")
		}
	}
	diff := stringutil.DiffMapAgainstStringSlice(installFlagsMap, sharedFlags)
	for _, d := range diff {
		delete(installFlagsMap, d)
	}
	// Build the final []string which consists of the flags with their
	// corresponding values
	final := []string{}
	for k, v := range installFlagsMap {
		// If we have a non-value flag we're going to put an empty string in
		// for v, this makes sure the slices match at the end
		if v == "" {
			final = append(final, k)
		} else {
			final = append(final, fmt.Sprintf("%s %s", k, v))
		}
	}
	return final
}

// FormatReplicaID returns a zero padded 12 character hex string.
func FormatReplicaID(num uint64) string {
	return fmt.Sprintf("%012x", num)
}

// BuildMKEFlags builds the mkeFlags []string consisting of mke installFlags
// that are shared with MSR.
func BuildMKEFlags(config *api.ClusterConfig) common.Flags {
	mkeUser := config.Spec.MSR.V2.InstallFlags.GetValue("--ucp-username")
	mkePass := config.Spec.MSR.V2.InstallFlags.GetValue("--ucp-password")

	if mkeUser == "" {
		mkeUser = config.Spec.MKE.AdminUsername
	}

	// Still empty? Default to admin.
	if mkeUser == "" {
		mkeUser = "admin"
	}

	if mkePass == "" {
		mkePass = config.Spec.MKE.AdminPassword
	}

	return common.Flags{
		fmt.Sprintf("--ucp-url=\"%s\"", mkeURLHost(config)),
		fmt.Sprintf("--ucp-username=\"%s\"", mkeUser),
		fmt.Sprintf("--ucp-password=\"%s\"", mkePass),
	}
}

func mkeURLHost(config *api.ClusterConfig) string {
	url, _ := config.Spec.MKEURL()
	// url.Host will be host:port when a port has been set
	return url.Host
}

// Destroy is functionally equivalent to a MSR destroy and is intended to
// remove any MSR containers and volumes that may have been started via the
// installer if it fails.
func Destroy(h *api.Host) error {
	// Remove containers
	log.Debugf("%s: Removing MSR containers", h)
	containersToRemove, err := h.ExecOutput(h.Configurer.DockerCommandf("ps -aq --filter name=dtr-"))
	if err != nil {
		return fmt.Errorf("failed to get MSR container list: %w", err)
	}
	if strings.TrimSpace(containersToRemove) == "" {
		log.Debugf("No MSR containers to remove")
	} else {
		containersToRemove = strings.Join(strings.Fields(containersToRemove), " ")
		if err := h.Exec(h.Configurer.DockerCommandf("rm -f %s", containersToRemove)); err != nil {
			return fmt.Errorf("failed to remove MSR containers: %w", err)
		}
	}

	// Remove volumes
	log.Debugf("%s: Removing MSR volumes", h)
	volumeOutput, err := h.ExecOutput(h.Configurer.DockerCommandf("volume ls -q"))
	if err != nil {
		return fmt.Errorf("failed to get MSR volume list: %w", err)
	}
	if strings.Trim(volumeOutput, " ") == "" {
		log.Debugf("No volumes in volume list")
	} else {
		// Iterate the volumeList and determine what we need to remove
		var volumesToRemove []string
		volumeList := strings.Fields(volumeOutput)
		for _, v := range volumeList {
			if strings.HasPrefix(v, "dtr-") {
				volumesToRemove = append(volumesToRemove, v)
			}
		}
		// Perform the removal
		if len(volumesToRemove) == 0 {
			log.Debugf("No MSR volumes to remove")
			return nil
		}
		volumes := strings.Join(volumesToRemove, " ")
		err = h.Exec(h.Configurer.DockerCommandf("volume rm -f %s", volumes))
		if err != nil {
			return fmt.Errorf("failed to remove MSR volumes: %w", err)
		}
	}
	return nil
}

var errMaxReplicaID = fmt.Errorf("max sequential msr replica id exceeded")

// AssignSequentialReplicaIDs goes through all the MSR hosts, finds the highest replica id and assigns sequential ones starting from that to all the hosts without replica ids.
func AssignSequentialReplicaIDs(c *api.ClusterConfig) error {
	msrHosts := c.Spec.MSRs()

	// find the largest replica id
	var maxReplicaID uint64
	err := msrHosts.Each(func(h *api.Host) error {
		if h.MSRMetadata == nil {
			h.MSRMetadata = &api.MSRMetadata{}
		}
		if h.MSRMetadata.MSR2.ReplicaID != "" {
			ri, err := strconv.ParseUint(h.MSRMetadata.MSR2.ReplicaID, 16, 48)
			if err != nil {
				return fmt.Errorf("%s: invalid MSR replicaID %q: %s", h, h.MSRMetadata.MSR2.ReplicaID, err)
			}
			if maxReplicaID < ri {
				maxReplicaID = ri
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to find max MSR replicaID: %w", err)
	}
	if maxReplicaID+uint64(len(msrHosts)) > 0xffffffffffff {
		return fmt.Errorf("%w: cluster already has replica id %012x which will overflow", errMaxReplicaID, maxReplicaID)
	}
	return msrHosts.Each(func(h *api.Host) error {
		if h.MSRMetadata.MSR2.ReplicaID == "" {
			maxReplicaID++
			h.MSRMetadata.MSR2.ReplicaID = FormatReplicaID(maxReplicaID)
		}
		return nil
	})

	return nil
}

// Cleanup accepts a list of msrHosts to remove all containers, volumes
// and networks on, it is intended to be used to uninstall MSR or cleanup
// a failed install.
func Cleanup(msrHosts []*api.Host, swarmLeader *api.Host) error {
	for _, h := range msrHosts {
		log.Debugf("%s: Destroying MSR host", h)
		err := Destroy(h)
		if err != nil {
			return fmt.Errorf("failed to run MSR destroy: %w", err)
		}
	}
	// Remove dtr-ol via the swarmLeader
	log.Infof("%s: Removing dtr-ol network", swarmLeader)
	if err := swarmLeader.Exec(swarmLeader.Configurer.DockerCommandf("network rm dtr-ol")); err != nil {
		return fmt.Errorf("failed to remove dtr-ol network: %w", err)
	}
	return nil
}

// WaitMSRNodeReady waits until MSR is up on the host.
func WaitMSRNodeReady(h *api.Host, port int) error {
	err := retry.Do(
		func() error {
			output, err := h.ExecOutput(h.Configurer.DockerCommandf("ps -q -f health=healthy -f name=dtr-nginx"))
			if err != nil || strings.TrimSpace(output) == "" {
				return fmt.Errorf("MSR nginx container not running: %w", err)
			}
			return nil
		},
		retry.DelayType(retry.CombineDelay(retry.FixedDelay, retry.RandomDelay)),
		retry.MaxJitter(time.Second*2),
		retry.Delay(time.Second*3),
		retry.Attempts(60),
	)
	if err != nil {
		return fmt.Errorf("retry limit exceeded: %w", err)
	}

	err = retry.Do(
		func() error {
			if err := h.CheckHTTPStatus(fmt.Sprintf("https://localhost:%d/_ping", port), 200); err != nil {
				return fmt.Errorf("MSR invalid ping response: %w", err)
			}

			return nil
		},
		retry.DelayType(retry.CombineDelay(retry.FixedDelay, retry.RandomDelay)),
		retry.MaxJitter(time.Second*2),
		retry.Delay(time.Second*3),
		retry.Attempts(120),
	)
	if err != nil {
		return fmt.Errorf("retry limit exceeded: %w", err)
	}
	return nil
}
