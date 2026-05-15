package phase

import (
	"testing"
)

func TestIsUninstallTimeout(t *testing.T) {
	t.Run("matches MKE timeout output", func(t *testing.T) {
		// MKE emits this at error level; it appears in Bootstrap's output string.
		output := "Uninstalling UCP took too long!\nThe following nodes are unable to uninstall within the timeout: abc123\n"
		if !isUninstallTimeout(output) {
			t.Errorf("expected isUninstallTimeout=true for MKE timeout output, got false")
		}
	})

	t.Run("does not match generic uninstall failure output", func(t *testing.T) {
		// "unable to cleanly uninstall UCP" is the fatal line — it should NOT
		// trigger dissolution on its own; it can appear for non-timeout reasons.
		output := "unable to cleanly uninstall UCP\n"
		if isUninstallTimeout(output) {
			t.Errorf("expected isUninstallTimeout=false for generic failure output, got true")
		}
	})

	t.Run("does not match empty output", func(t *testing.T) {
		if isUninstallTimeout("") {
			t.Errorf("expected isUninstallTimeout=false for empty output, got true")
		}
	})
}
