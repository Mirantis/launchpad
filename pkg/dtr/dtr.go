package dtr

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/util"
	log "github.com/sirupsen/logrus"
)

// ErrorNoSuchObject mocks the "No such object" error returned from docker
// engine that is returned when a container or volume is listed but nothing
// matching that object is found

// CollectDtrFacts gathers the current status of the installed DTR setup
func CollectDtrFacts(dtrHost *api.Host) (*api.DtrMetadata, error) {
	rethinkdbContainerID, err := dtrHost.ExecWithOutput(dtrHost.Configurer.DockerCommandf(`ps -aq --filter name=dtr-rethinkdb`))
	if err != nil {
		return nil, err
	}
	if rethinkdbContainerID == "" {
		return &api.DtrMetadata{Installed: false}, nil
	}

	version, err := dtrHost.ExecWithOutput(dtrHost.Configurer.DockerCommandf(`inspect %s --format '{{ index .Config.Labels "com.docker.dtr.version"}}'`, rethinkdbContainerID))
	if err != nil {
		return nil, err
	}
	replicaID, err := dtrHost.ExecWithOutput(dtrHost.Configurer.DockerCommandf(`inspect %s --format '{{ index .Config.Labels "com.docker.dtr.replica"}}'`, rethinkdbContainerID))
	if err != nil {
		return nil, err
	}
	if version == "" || replicaID == "" {
		// If we failed to obtain either label then this DTR version does not
		// support the version Config.Labels or something else may have gone
		// wrong, attempt to pull these details with the old method
		output, err := dtrHost.ExecWithOutput(dtrHost.Configurer.DockerCommandf(`inspect %s --format '{{ index .Config.Labels "com.docker.compose.project"}}'`, rethinkdbContainerID))
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
	imagename, err := dtrHost.ExecWithOutput(dtrHost.Configurer.DockerCommandf(`inspect %s --format '{{ .Config.Image }}'`, rethinkdbContainerID))
	if err == nil {
		repo := imagename[:strings.LastIndexByte(imagename, '/')]
		bootstrapimage = fmt.Sprintf("%s/dtr:%s", repo, version)
	}

	dtrMeta := &api.DtrMetadata{
		Installed:               true,
		InstalledVersion:        version,
		InstalledBootstrapImage: bootstrapimage,
		DtrLeaderReplicaID:      replicaID,
	}
	return dtrMeta, nil
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

// SequentialReplicaID returns a replica id for a given int intended to be used
// to construct a sequential number of replicas up to a value of 9
func SequentialReplicaID(replicaInt int) string {
	replicaPrefix := "00000000000"
	return fmt.Sprintf("%s%d", replicaPrefix, replicaInt)
}

// BuildUcpFlags builds the ucpFlags []string consisting of ucp installFlags
// that are shared with DTR
func BuildUcpFlags(config *api.ClusterConfig) []string {
	return []string{
		fmt.Sprintf("--ucp-url %s", GetUcpURL(config)),
		fmt.Sprintf("--ucp-username %s", util.GetInstallFlagValue(config.Spec.Ucp.InstallFlags, "--admin-username")),
		fmt.Sprintf("--ucp-password '%s'", util.GetInstallFlagValue(config.Spec.Ucp.InstallFlags, "--admin-password")),
	}
}

// GetUcpURL builds the ucp url from the --san flag or from the swarmLeader
func GetUcpURL(config *api.ClusterConfig) string {
	if config.Spec.Dtr == nil {
		return ""
	}

	dtrUcpURLFlag := util.GetInstallFlagValue(config.Spec.Dtr.InstallFlags, "--ucp-url")
	if dtrUcpURLFlag != "" {
		// If the --ucp-url flag has been set use that instead
		return dtrUcpURLFlag
	}

	sanFlag := util.GetInstallFlagValue(config.Spec.Ucp.InstallFlags, "--san")
	if sanFlag != "" {
		// If a san value has been provided by the user, use that as ucp-url
		return sanFlag
	}
	// Else get the swarmLeader address and append the set --controller-port if
	// it's non-default
	controllerPort := util.GetInstallFlagValue(config.Spec.Ucp.InstallFlags, "--controller-port")
	if controllerPort == "" {
		controllerPort = "443"
	}
	return fmt.Sprintf("%s:%s", config.Spec.SwarmLeader().Address, controllerPort)
}

// Destroy is functionally equivalent to a DTR destroy and is intended to
// remove any DTR containers and volumes that may have been started via the
// installer if it fails
func Destroy(dtrHost *api.Host) error {
	// Remove containers
	log.Debugf("%s: Removing DTR containers", dtrHost.Address)
	containersToRemove, err := dtrHost.ExecWithOutput(dtrHost.Configurer.DockerCommandf("ps -aq --filter name=dtr-"))
	if err != nil {
		return err
	}
	if strings.TrimSpace(containersToRemove) == "" {
		log.Debugf("No DTR containers to remove")
	} else {
		containersToRemove = strings.Join(strings.Fields(containersToRemove), " ")
		if err := dtrHost.Exec(dtrHost.Configurer.DockerCommandf("rm -f %s", containersToRemove)); err != nil {
			return err
		}
	}

	// Remove volumes
	log.Debugf("%s: Removing DTR volumes", dtrHost.Address)
	volumeOutput, err := dtrHost.ExecWithOutput(dtrHost.Configurer.DockerCommandf("volume ls -q"))
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
			log.Debugf("No DTR volumes to remove")
			return nil
		}
		volumes := strings.Join(volumesToRemove, " ")
		err = dtrHost.Exec(dtrHost.Configurer.DockerCommandf("volume rm -f %s", volumes))
		if err != nil {
			return err
		}
	}
	return nil
}

// CleanupDtrs accepts a list of dtrHosts to remove all containers, volumes
// and networks on, it is intended to be used to uninstall DTR or cleanup
// a failed install
func CleanupDtrs(dtrHosts []*api.Host, swarmLeader *api.Host) error {
	for _, h := range dtrHosts {
		log.Debugf("%s: Destroying DTR host", h.Address)
		err := Destroy(h)
		if err != nil {
			return fmt.Errorf("failed to run DTR destroy: %s", err)
		}
	}
	// Remove dtr-ol via the swarmLeader
	log.Infof("%s: Removing dtr-ol network", swarmLeader.Address)
	swarmLeader.Exec(swarmLeader.Configurer.DockerCommandf("network rm dtr-ol"))
	return nil
}
