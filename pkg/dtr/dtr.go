package dtr

import (
	"errors"
	"fmt"
	"strings"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta2"
	"github.com/Mirantis/mcc/pkg/util"
)

// Details defines DTR host details
type Details struct {
	Version   string
	ReplicaID string
}

// ErrorNoSuchObject mocks the "No such object" error returned from docker
// engine that is returned when a container or volume is listed but nothing
// matching that object is found
var ErrorNoSuchObject error = errors.New("No such object")

// CollectDtrFacts gathers the current status of the installed DTR setup
func CollectDtrFacts(dtrLeader *api.Host) (*api.DtrMetadata, error) {
	var details *Details
	details, err := GetDTRDetails(dtrLeader)
	if err != nil {
		if errors.Is(err, ErrorNoSuchObject) {
			return &api.DtrMetadata{Installed: false}, nil
		}
		return nil, err
	}

	dtrMeta := &api.DtrMetadata{
		Installed:          true,
		InstalledVersion:   details.Version,
		DtrLeaderReplicaID: details.ReplicaID,
	}
	return dtrMeta, nil
}

// GetDTRDetails returns a struct containing the DTR version and replica ID from
// the host it's executed on
func GetDTRDetails(dtrHost *api.Host) (*Details, error) {
	rethinkdbContainerID, err := dtrHost.ExecWithOutput(dtrHost.Configurer.DockerCommandf(`ps -aq --filter name=dtr-rethinkdb`))
	if err != nil {
		return nil, err
	}
	if rethinkdbContainerID == "" {
		return nil, ErrorNoSuchObject
	}
	output, err := dtrHost.ExecWithOutput(dtrHost.Configurer.DockerCommandf(`inspect %s --format '{{ index .Config.Labels "com.docker.compose.project"}}'`, rethinkdbContainerID))
	if err != nil {
		return nil, err
	}

	outputFields := strings.Fields(output)
	details := &Details{
		Version:   outputFields[3],
		ReplicaID: strings.Trim(outputFields[len(outputFields)-1], ")"),
	}

	return details, nil
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

// GetBootstrapVersion gets the version of bootstrapper image from the specified
// host
func GetBootstrapVersion(dtrHost *api.Host, config *api.ClusterConfig) (string, error) {
	output, err := dtrHost.ExecWithOutput(dtrHost.Configurer.DockerCommandf(`image inspect %s --format '{{.RepoTags}}'`, config.Spec.Dtr.GetBootstrapperImage()))
	if err != nil {
		return "", err
	}
	outputSplit := strings.Split(output, ":")
	if len(outputSplit) != 2 {
		return "", fmt.Errorf("unexpected length of DTR bootstrapper image RepoTags")
	}
	version := strings.Trim(outputSplit[1], "]")
	return version, nil
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
