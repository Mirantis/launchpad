package phase

import (
	"testing"

	"github.com/Mirantis/mcc/pkg/docker/hub"
	"github.com/hashicorp/go-version"
)

func TestMKEVersionRetrieve(t *testing.T) {
	msrv, err := hub.LatestTag(hub.RegistryDockerHub, "mirantis", "ucp", false)
	if err != nil {
		t.Fatalf("Failed to retrieve MKE3 version: %s", err.Error())
	}

	if msrv == "" {
		t.Fatal("Empty MKE3 version retrieved from registry")
	}

	msrV, err := version.NewVersion(msrv)
	if err != nil {
		t.Errorf("invalid MKE3 version response: %s", err.Error())
	}

	msrTargetV, _ := version.NewVersion("3.0.0")

	if !msrV.GreaterThan(msrTargetV) {
		t.Errorf("Failed to detect newer version MKE3: %s (%s)", msrv, msrTargetV.String())
	}
}

func TestMSR2VersionRetrieve(t *testing.T) {
	msrv, err := hub.LatestTag(hub.RegistryDockerHub, "mirantis", "dtr", false)
	if err != nil {
		t.Fatalf("Failed to retrieve MSR2 version: %s", err.Error())
	}

	if msrv == "" {
		t.Fatal("Empty MSR2 version retrieved from registry")
	}

	msrV, err := version.NewVersion(msrv)
	if err != nil {
		t.Errorf("invalid MSR2 version response: %s", err.Error())
	}

	msrTargetV, _ := version.NewVersion("2.9.0")

	if !msrV.GreaterThan(msrTargetV) {
		t.Errorf("Failed to detect newer version MSR2: %s (%s)", msrv, msrTargetV.String())
	}
}

func TestMSR3VersionRetrieve(t *testing.T) {
	msrv, err := hub.LatestTag(hub.RegistryMirantis, "msr", "msr-api", false)
	if err != nil {
		t.Fatalf("Failed to retrieve MSR3 version: %s", err.Error())
	}

	if msrv == "" {
		t.Fatal("Empty MSR3 version retrieved from registry")
	}

	msrV, err := version.NewVersion(msrv)
	if err != nil {
		t.Errorf("invalid MSR3 version response: %s", err.Error())
	}

	msrTargetV, _ := version.NewVersion("3.0.0")

	if !msrV.GreaterThan(msrTargetV) {
		t.Errorf("Failed to detect newer version MSR3: %s (%s)", msrv, msrTargetV.String())
	}
}
