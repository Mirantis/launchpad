package msr

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/util"
	log "github.com/sirupsen/logrus"
)

// CollectFacts gathers the current status of the installed MSR setup
func CollectFacts(h *api.Host) (*api.MSRMetadata, error) {
	rethinkdbContainerID, err := h.ExecWithOutput(h.Configurer.DockerCommandf(`ps -aq --filter name=dtr-rethinkdb`))
	if err != nil {
		return nil, err
	}
	if rethinkdbContainerID == "" {
		return &api.MSRMetadata{Installed: false}, nil
	}

	version, err := h.ExecWithOutput(h.Configurer.DockerCommandf(`inspect %s --format '{{ index .Config.Labels "com.docker.dtr.version"}}'`, rethinkdbContainerID))
	if err != nil {
		return nil, err
	}
	replicaID, err := h.ExecWithOutput(h.Configurer.DockerCommandf(`inspect %s --format '{{ index .Config.Labels "com.docker.dtr.replica"}}'`, rethinkdbContainerID))
	if err != nil {
		return nil, err
	}
	if version == "" || replicaID == "" {
		// If we failed to obtain either label then this MSR version does not
		// support the version Config.Labels or something else may have gone
		// wrong, attempt to pull these details with the old method
		output, err := h.ExecWithOutput(h.Configurer.DockerCommandf(`inspect %s --format '{{ index .Config.Labels "com.docker.compose.project"}}'`, rethinkdbContainerID))
		if err != nil {
			return nil, err
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
	imagename, err := h.ExecWithOutput(h.Configurer.DockerCommandf(`inspect %s --format '{{ .Config.Image }}'`, rethinkdbContainerID))
	if err == nil {
		repo := imagename[:strings.LastIndexByte(imagename, '/')]
		bootstrapimage = fmt.Sprintf("%s/dtr:%s", repo, version)
	}

	msrMeta := &api.MSRMetadata{
		Installed:               true,
		InstalledVersion:        version,
		InstalledBootstrapImage: bootstrapimage,
		MSRLeaderReplicaID:      replicaID,
	}
	return msrMeta, nil
}

// PluckSharedInstallFlags plucks the shared flag values between install and
// shared and returns a slice of flags and their values
// FIXME(squizzi): There's probably a better way to do this, this is a bit
// overkill
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
	diff := util.DiffMapAgainstStringSlice(installFlagsMap, sharedFlags)
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

// BuildMKEFlags builds the mkeFlags []string consisting of mke installFlags
// that are shared with MSR
func BuildMKEFlags(config *api.ClusterConfig) []string {
	var mkeUser string
	var mkePass string

	if config.Spec.MSR != nil {
		mkeUser = config.Spec.MSR.InstallFlags.GetValue("--ucp-username")
		mkePass = config.Spec.MSR.InstallFlags.GetValue("--ucp-password")
	}

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

	return []string{
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
// installer if it fails
func Destroy(h *api.Host) error {
	// Remove containers
	log.Debugf("%s: Removing MSR containers", h)
	containersToRemove, err := h.ExecWithOutput(h.Configurer.DockerCommandf("ps -aq --filter name=dtr-"))
	if err != nil {
		return err
	}
	if strings.TrimSpace(containersToRemove) == "" {
		log.Debugf("No MSR containers to remove")
	} else {
		containersToRemove = strings.Join(strings.Fields(containersToRemove), " ")
		if err := h.Exec(h.Configurer.DockerCommandf("rm -f %s", containersToRemove)); err != nil {
			return err
		}
	}

	// Remove volumes
	log.Debugf("%s: Removing MSR volumes", h)
	volumeOutput, err := h.ExecWithOutput(h.Configurer.DockerCommandf("volume ls -q"))
	if err != nil {
		return err
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
			return err
		}
	}
	return nil
}

// Cleanup accepts a list of msrHosts to remove all containers, volumes
// and networks on, it is intended to be used to uninstall MSR or cleanup
// a failed install
func Cleanup(msrHosts []*api.Host, swarmLeader *api.Host) error {
	for _, h := range msrHosts {
		log.Debugf("%s: Destroying MSR host", h)
		err := Destroy(h)
		if err != nil {
			return fmt.Errorf("failed to run MSR destroy: %s", err)
		}
	}
	// Remove dtr-ol via the swarmLeader
	log.Infof("%s: Removing dtr-ol network", swarmLeader)
	swarmLeader.Exec(swarmLeader.Configurer.DockerCommandf("network rm dtr-ol"))
	return nil
}
