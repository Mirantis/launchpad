package mkex

import (
	"testing"

	"github.com/k0sproject/rig"
)

func TestIsMKEOSVersion(t *testing.T) {
	osvIsMKE := rig.OSVersion{ExtraFields: map[string]string{}}
	osvIsMKE.ExtraFields[mkexDetectExtraFieldKey] = "test"

	osvIsNotMKE := rig.OSVersion{}

	if !isMKExOSVersion(osvIsMKE) {
		t.Error("MKEx detector failed to detect an MKEx OSVersion")
	}
	if isMKExOSVersion(osvIsNotMKE) {
		t.Error("MKEx detector detected an MKEx OSVersion when it shouldn't have")
	}
}
