package phase

import (
	"bytes"
	"strings"
	"testing"

	commonconfig "github.com/Mirantis/launchpad/pkg/product/common/config"
	mkeconfig "github.com/Mirantis/launchpad/pkg/product/mke/config"
	"github.com/k0sproject/rig"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestValidateFactsMKEVersionJumpFail(t *testing.T) {
	phase := ValidateFacts{}
	phase.Config = &mkeconfig.ClusterConfig{
		Spec: &mkeconfig.ClusterSpec{
			MKE: mkeconfig.MKEConfig{
				Metadata: &mkeconfig.MKEMetadata{
					Installed:        true,
					InstalledVersion: "3.1.1",
				},
				Version: "3.3.3-tp9",
			},
		},
	}
	require.ErrorContains(t, phase.validateMKEVersionJump(), "can't upgrade MKE directly from 3.1.1 to 3.3.3-tp9 - need to upgrade to 3.2 first")
}

func TestValidateFactsMKEVersionJumpDowngradeFail(t *testing.T) {
	phase := ValidateFacts{}
	phase.Config = &mkeconfig.ClusterConfig{
		Spec: &mkeconfig.ClusterSpec{
			MKE: mkeconfig.MKEConfig{
				Metadata: &mkeconfig.MKEMetadata{
					Installed:        true,
					InstalledVersion: "3.3.3-tp9",
				},
				Version: "3.2.8",
			},
		},
	}
	require.ErrorContains(t, phase.validateMKEVersionJump(), "can't downgrade MKE 3.3.3-tp9 to 3.2.8")
}

func TestValidateFactsMKEVersionJumpSuccess(t *testing.T) {
	phase := ValidateFacts{}
	phase.Config = &mkeconfig.ClusterConfig{
		Spec: &mkeconfig.ClusterSpec{
			MKE: mkeconfig.MKEConfig{
				Metadata: &mkeconfig.MKEMetadata{
					Installed:        true,
					InstalledVersion: "3.1.1",
				},
				Version: "3.2.8",
			},
		},
	}
	require.NoError(t, phase.validateMKEVersionJump())
}

func TestValidateFactsMSRVersionJumpFail(t *testing.T) {
	phase := ValidateFacts{}
	phase.Config = &mkeconfig.ClusterConfig{
		Spec: &mkeconfig.ClusterSpec{
			Hosts: []*mkeconfig.Host{
				{Role: "msr", MSRMetadata: &mkeconfig.MSRMetadata{
					Installed:        true,
					InstalledVersion: "2.6.4",
				}},
			},
			MSR: &mkeconfig.MSRConfig{
				Version: "2.8.4",
			},
		},
	}
	require.ErrorContains(t, phase.validateMSRVersionJump(), "can't upgrade MSR directly from 2.6.4 to 2.8.4 - need to upgrade to 2.7 first")
}
func TestValidateFactsMSRVersionJumpDowngradeFail(t *testing.T) {
	phase := ValidateFacts{}
	phase.Config = &mkeconfig.ClusterConfig{
		Spec: &mkeconfig.ClusterSpec{
			Hosts: []*mkeconfig.Host{
				{Role: "msr", MSRMetadata: &mkeconfig.MSRMetadata{
					Installed:        true,
					InstalledVersion: "2.8.4",
				}},
			},
			MSR: &mkeconfig.MSRConfig{
				Version: "2.7.6",
			},
		},
	}
	require.ErrorContains(t, phase.validateMSRVersionJump(), "can't downgrade MSR 2.8.4 to 2.7.6")
}

func TestValidateFactsMSRVersionJumpSuccess(t *testing.T) {
	phase := ValidateFacts{}
	phase.Config = &mkeconfig.ClusterConfig{
		Spec: &mkeconfig.ClusterSpec{
			Hosts: []*mkeconfig.Host{
				{Role: "msr", MSRMetadata: &mkeconfig.MSRMetadata{
					Installed:        true,
					InstalledVersion: "2.6.8",
				}},
			},
			MSR: &mkeconfig.MSRConfig{
				Version: "2.7.1",
			},
		},
	}
	require.NoError(t, phase.validateMSRVersionJump())
}

func TestValidateFactsValidateDataPlane(t *testing.T) {
	phase := ValidateFacts{}
	phase.Config = &mkeconfig.ClusterConfig{
		Spec: &mkeconfig.ClusterSpec{
			MKE: mkeconfig.MKEConfig{
				InstallFlags: []string{
					"--foo",
					"--calico-vxlan=true",
				},
				Metadata: &mkeconfig.MKEMetadata{
					Installed: true,
					VXLAN:     false,
				},
			},
		},
	}

	// Test meta-vxlan: false, --calico-vxlan=true
	require.ErrorContains(t, phase.validateDataPlane(), "calico configured with IPIP, can't automatically change to VXLAN")

	// Test meta-vxlan: false, --calico-vxlan (should evaluate to true)
	phase.Config.Spec.MKE.InstallFlags = []string{
		"--calico-vxlan",
	}
	require.ErrorContains(t, phase.validateDataPlane(), "calico configured with IPIP, can't automatically change to VXLAN")

	// Test with meta-vxlan: true, --calico-vxlan true
	phase.Config.Spec.MKE.Metadata.VXLAN = true
	require.NoError(t, phase.validateDataPlane())

	// Test with meta-vxlan: true, --calico-vxlan false
	phase.Config.Spec.MKE.InstallFlags = []string{
		"--calico-vxlan=false",
	}
	require.ErrorContains(t, phase.validateDataPlane(), "calico configured with VXLAN, can't automatically change to IPIP")

	// Test with meta-vxlan: false, --calico-vxlan false
	phase.Config.Spec.MKE.Metadata.VXLAN = false
	require.NoError(t, phase.validateDataPlane())
}

func TestValidateFactsPopulateSan(t *testing.T) {
	phase := ValidateFacts{}
	phase.Config = &mkeconfig.ClusterConfig{
		Spec: &mkeconfig.ClusterSpec{
			Hosts: mkeconfig.Hosts{
				&mkeconfig.Host{Connection: rig.Connection{SSH: &rig.SSH{Address: "10.0.0.1"}}, Role: "manager"},
				&mkeconfig.Host{Connection: rig.Connection{SSH: &rig.SSH{Address: "10.0.0.2"}}, Role: "manager"},
				&mkeconfig.Host{Connection: rig.Connection{SSH: &rig.SSH{Address: "10.0.0.3"}}, Role: "worker"},
			},
			MCR: commonconfig.MCRConfig{
				Channel: "stable-25.0",
			},
			MKE: mkeconfig.MKEConfig{
				Version:  "3.6.0",
				Metadata: &mkeconfig.MKEMetadata{},
				InstallFlags: commonconfig.Flags{
					"--foo",
				},
			},
		},
	}
	require.NoError(t, phase.Run())
	var sans []string

	for _, v := range phase.Config.Spec.MKE.InstallFlags {
		if strings.HasPrefix(v, "--san") {
			sans = append(sans, v)
		}
	}

	require.Len(t, phase.Config.Spec.MKE.InstallFlags, 3, "InstallFlags should be --foo plus two --san entries for the two managers")
	require.Len(t, sans, 2)

	require.Equal(t, "--san=10.0.0.1", sans[0])
	require.Equal(t, "--san=10.0.0.2", sans[1])
}

func TestValidateFactsDontPopulateSan(t *testing.T) {
	phase := ValidateFacts{}
	phase.Config = &mkeconfig.ClusterConfig{
		Spec: &mkeconfig.ClusterSpec{
			Hosts: mkeconfig.Hosts{
				&mkeconfig.Host{Connection: rig.Connection{SSH: &rig.SSH{Address: "10.0.0.1"}}, Role: "manager"},
				&mkeconfig.Host{Connection: rig.Connection{SSH: &rig.SSH{Address: "10.0.0.2"}}, Role: "manager"},
				&mkeconfig.Host{Connection: rig.Connection{SSH: &rig.SSH{Address: "10.0.0.3"}}, Role: "worker"},
			},
			MCR: commonconfig.MCRConfig{
				Channel: "stable-25.0",
			},
			MKE: mkeconfig.MKEConfig{
				Version:  "3.6.0",
				Metadata: &mkeconfig.MKEMetadata{},
				InstallFlags: commonconfig.Flags{
					"--foo",
					"--san foofoo",
				},
			},
		},
	}
	require.NoError(t, phase.Run())
	var sans []string

	for _, v := range phase.Config.Spec.MKE.InstallFlags {
		if strings.HasPrefix(v, "--san") {
			sans = append(sans, v)
		}
	}

	require.Len(t, sans, 1, "Run must not add manager SANs when --san is already present")
	require.Equal(t, "--san foofoo", sans[0])
}

func TestValidateInvalidMCRConfig(t *testing.T) {
	phase := ValidateFacts{}
	phase.Config = &mkeconfig.ClusterConfig{
		Spec: &mkeconfig.ClusterSpec{
			Hosts: mkeconfig.Hosts{
				&mkeconfig.Host{Connection: rig.Connection{SSH: &rig.SSH{Address: "10.0.0.1"}}, Role: "manager"},
			},
		},
	}

	require.Error(t, phase.Run(), "MCR version validated an invalid config")
}

func makePhaseWithPodCIDR(podCIDR string, swarmPools ...string) ValidateFacts {
	p := ValidateFacts{}
	installFlags := commonconfig.Flags{"--san=10.0.0.1"}
	if podCIDR != "" {
		installFlags = append(installFlags, "--pod-cidr "+podCIDR)
	}
	swarmFlags := commonconfig.Flags{}
	for _, swarmPool := range swarmPools {
		if swarmPool != "" {
			swarmFlags = append(swarmFlags, "--default-addr-pool "+swarmPool)
		}
	}
	p.Config = &mkeconfig.ClusterConfig{
		Spec: &mkeconfig.ClusterSpec{
			Hosts: mkeconfig.Hosts{
				&mkeconfig.Host{Connection: rig.Connection{SSH: &rig.SSH{Address: "10.0.0.1"}}, Role: "manager"},
			},
			MCR: commonconfig.MCRConfig{
				Channel:           "stable-29.4",
				SwarmInstallFlags: swarmFlags,
				Metadata:          &commonconfig.MCRMetadata{},
			},
			MKE: mkeconfig.MKEConfig{
				Version:      "3.8.2",
				Metadata:     &mkeconfig.MKEMetadata{},
				InstallFlags: installFlags,
			},
		},
	}
	return p
}

func TestValidatePodCIDROverlapsDefaultPool(t *testing.T) {
	// 10.0.0.0/16 is a subnet of the default Swarm pool 10.0.0.0/8 — must fail.
	p := makePhaseWithPodCIDR("10.0.0.0/16", "")
	err := p.validatePodCIDR()
	require.Error(t, err)
	require.ErrorIs(t, err, errInvalidPodCIDR)
	require.ErrorContains(t, err, "10.0.0.0/16")
	require.ErrorContains(t, err, "10.0.0.0/8")
}

func TestValidatePodCIDRNoOverlap(t *testing.T) {
	// 192.168.0.0/16 does not overlap with 10.0.0.0/8.
	p := makePhaseWithPodCIDR("192.168.0.0/16", "")
	require.NoError(t, p.validatePodCIDR())
}

func TestValidatePodCIDRAbsent(t *testing.T) {
	// No --pod-cidr flag — validation must be a no-op.
	p := makePhaseWithPodCIDR("", "")
	require.NoError(t, p.validatePodCIDR())
}

func TestValidatePodCIDRCustomSwarmPool(t *testing.T) {
	// --default-addr-pool overrides the compiled-in default.
	// 10.0.0.0/16 would overlap 10.0.0.0/8 but not 172.16.0.0/12.
	p := makePhaseWithPodCIDR("10.0.0.0/16", "172.16.0.0/12")
	require.NoError(t, p.validatePodCIDR())
}

func TestValidatePodCIDRCustomSwarmPoolOverlap(t *testing.T) {
	// Custom pool 192.168.0.0/16 overlaps with pod-cidr 192.168.1.0/24.
	p := makePhaseWithPodCIDR("192.168.1.0/24", "192.168.0.0/16")
	err := p.validatePodCIDR()
	require.Error(t, err)
	require.ErrorIs(t, err, errInvalidPodCIDR)
}

func TestValidatePodCIDROverlapsLaterSwarmPool(t *testing.T) {
	// docker swarm init accepts --default-addr-pool repeated. The pod-cidr
	// overlaps only the SECOND pool — validation must still fail, proving every
	// pool is checked and not just the first.
	p := makePhaseWithPodCIDR("172.16.0.0/16", "192.168.0.0/16", "172.16.0.0/12")
	err := p.validatePodCIDR()
	require.Error(t, err)
	require.ErrorIs(t, err, errInvalidPodCIDR)
	require.ErrorContains(t, err, "172.16.0.0/12")
}

func TestValidatePodCIDRMultipleSwarmPoolsNoOverlap(t *testing.T) {
	// Multiple pools, none overlapping the pod-cidr — validation passes.
	p := makePhaseWithPodCIDR("10.96.0.0/16", "192.168.0.0/16", "172.16.0.0/12")
	require.NoError(t, p.validatePodCIDR())
}

func TestValidatePodCIDRContainsSwarmPool(t *testing.T) {
	// Reverse containment: the pod-cidr is BROADER than the swarm pool, so
	// podNet.Contains(swarmNet.IP) is the deciding operand. Guards the second
	// half of the overlap check, which the other overlap tests never exercise.
	p := makePhaseWithPodCIDR("172.16.0.0/12", "172.20.0.0/16")
	err := p.validatePodCIDR()
	require.Error(t, err)
	require.ErrorIs(t, err, errInvalidPodCIDR)
}

func TestValidatePodCIDRMalformedPodCIDR(t *testing.T) {
	// A malformed --pod-cidr must fail fast with a parse error, not panic or pass.
	p := makePhaseWithPodCIDR("not-a-cidr", "")
	err := p.validatePodCIDR()
	require.Error(t, err)
	require.ErrorIs(t, err, errInvalidPodCIDR)
	require.ErrorContains(t, err, "cannot parse --pod-cidr")
}

func TestValidatePodCIDRMalformedSwarmPool(t *testing.T) {
	// A malformed --default-addr-pool must fail with a distinct parse error.
	p := makePhaseWithPodCIDR("10.244.0.0/16", "garbage")
	err := p.validatePodCIDR()
	require.Error(t, err)
	require.ErrorIs(t, err, errInvalidPodCIDR)
	require.ErrorContains(t, err, "cannot parse Swarm address pool")
}

// --- swarm overlay address pool (PRODENG-3642) -------------------------------

// podCIDRPhase builds a ValidateFacts phase for the pod CIDR checks. A nil
// livePools means no existing swarm was discovered, i.e. a first install.
func podCIDRPhase(podCIDR string, configuredPools, livePools []string) ValidateFacts {
	installFlags := commonconfig.Flags{}
	if podCIDR != "" {
		installFlags = append(installFlags, "--pod-cidr="+podCIDR)
	}

	swarmFlags := commonconfig.Flags{}
	for _, pool := range configuredPools {
		swarmFlags = append(swarmFlags, "--default-addr-pool="+pool)
	}

	phase := ValidateFacts{}
	phase.Config = &mkeconfig.ClusterConfig{
		Spec: &mkeconfig.ClusterSpec{
			MKE: mkeconfig.MKEConfig{InstallFlags: installFlags},
			MCR: commonconfig.MCRConfig{
				SwarmInstallFlags: swarmFlags,
				Metadata:          &commonconfig.MCRMetadata{SwarmDefaultAddrPool: livePools},
			},
		},
	}

	return phase
}

// captureLogs returns everything fn logs, so warn-only behaviour is observable.
func captureLogs(t *testing.T, fn func()) string {
	t.Helper()

	var buf bytes.Buffer
	logger := logrus.StandardLogger()
	originalOut, originalLevel := logger.Out, logger.GetLevel()
	logger.SetOutput(&buf)
	logger.SetLevel(logrus.DebugLevel)

	t.Cleanup(func() {
		logger.SetOutput(originalOut)
		logger.SetLevel(originalLevel)
	})

	fn()

	return buf.String()
}

// The reported bug: on an existing swarm the configured pool is discarded by
// InitSwarm, so validating against it hides a live overlap. The overlap must be
// reported, and must not block a cluster that already runs this way.
func TestValidatePodCIDRWarnsOnLiveOverlapHiddenByConfiguredPool(t *testing.T) {
	phase := podCIDRPhase("10.244.0.0/16", []string{"10.0.0.0/16"}, []string{"10.0.0.0/8"})

	var err error
	out := captureLogs(t, func() { err = phase.validatePodCIDR() })

	require.NoError(t, err, "the pool of an existing swarm cannot be changed, so this must not fail the run")
	require.Contains(t, out, "--pod-cidr 10.244.0.0/16 overlaps the overlay address pool 10.0.0.0/8 of the existing swarm")
	require.Contains(t, out, "cannot resolve this on a running cluster")
}

// The live pool takes precedence: this configuration would pass on its own.
func TestValidatePodCIDRPrefersLivePoolOverConfiguredPool(t *testing.T) {
	phase := podCIDRPhase("10.244.0.0/16", []string{"10.10.0.0/16"}, []string{"10.244.0.0/16"})

	var err error
	out := captureLogs(t, func() { err = phase.validatePodCIDR() })

	require.NoError(t, err)
	require.Contains(t, out, "of the existing swarm")
}

// A swarm may hold several pools; all of them are checked.
func TestValidatePodCIDRChecksEveryLivePool(t *testing.T) {
	phase := podCIDRPhase("10.244.0.0/16", nil, []string{"10.10.0.0/16", "10.244.0.0/16"})

	var err error
	out := captureLogs(t, func() { err = phase.validatePodCIDR() })

	require.NoError(t, err)
	require.Contains(t, out, "overlay address pool 10.244.0.0/16")
}

func TestValidatePodCIDRSilentWhenLivePoolDoesNotOverlap(t *testing.T) {
	phase := podCIDRPhase("10.244.0.0/16", nil, []string{"10.10.0.0/16"})

	var err error
	out := captureLogs(t, func() { err = phase.validatePodCIDR() })

	require.NoError(t, err)
	require.NotContains(t, out, "of the existing swarm")
}

// Metadata is populated from yaml and can be absent when a config is built by
// hand; the check must fall back to the configured pool rather than panic.
func TestValidatePodCIDRToleratesMissingMCRMetadata(t *testing.T) {
	phase := podCIDRPhase("10.244.0.0/16", nil, nil)
	phase.Config.Spec.MCR.Metadata = nil

	require.ErrorContains(t, phase.validatePodCIDR(), "10.0.0.0/8")
}

// A pool configured on an existing swarm changes nothing on the hosts, which is
// what makes the cluster stop matching its configuration.
func TestWarnSwarmAddrPoolDivergenceReportsInertSetting(t *testing.T) {
	phase := podCIDRPhase("", []string{"10.0.0.0/16"}, []string{"10.0.0.0/8"})

	out := captureLogs(t, phase.warnSwarmAddrPoolDivergence)

	require.Contains(t, out, "sets --default-addr-pool 10.0.0.0/16 but the existing swarm allocates overlay networks from 10.0.0.0/8")
	require.Contains(t, out, "has no effect here")
}

// The ordinary case: no pool configured and a swarm on the default. Comparing an
// empty configuration against the fallback would warn on every apply of every
// cluster that never set the flag.
func TestWarnSwarmAddrPoolDivergenceSilentWhenPoolNotConfigured(t *testing.T) {
	phase := podCIDRPhase("", nil, []string{"10.0.0.0/8"})
	require.Empty(t, captureLogs(t, phase.warnSwarmAddrPoolDivergence))
}

func TestWarnSwarmAddrPoolDivergenceSilentWhenConfigMatchesSwarm(t *testing.T) {
	phase := podCIDRPhase("", []string{"10.10.0.0/16"}, []string{"10.10.0.0/16"})
	require.Empty(t, captureLogs(t, phase.warnSwarmAddrPoolDivergence))
}

// On a first install there is no swarm to diverge from and the flags will be
// applied, so there is nothing to report.
func TestWarnSwarmAddrPoolDivergenceSilentOnFirstInstall(t *testing.T) {
	phase := podCIDRPhase("", []string{"10.10.0.0/16"}, nil)
	require.Empty(t, captureLogs(t, phase.warnSwarmAddrPoolDivergence))
}

func TestWarnSwarmAddrPoolDivergenceToleratesMissingMCRMetadata(t *testing.T) {
	phase := podCIDRPhase("", []string{"10.10.0.0/16"}, nil)
	phase.Config.Spec.MCR.Metadata = nil

	require.Empty(t, captureLogs(t, phase.warnSwarmAddrPoolDivergence))
}
