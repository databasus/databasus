package upgrade

import "testing"

func TestCombinedStorageRetirement(t *testing.T) {
	t.Parallel()
	t.Run("preserves both workspaces on refusal and upgrades directly after removal on 3.67.0", func(t *testing.T) {
		t.Parallel()
		runRetirementScenario(t, retirementScenario{
			image:    googleDriveReleaseImage,
			guidance: "Google Drive and rclone storages prevent this upgrade. Roll back to Databasus 3.67.0 and remove all Google Drive and rclone storages before upgrading, or continue using 3.67.0.",
			settings: []any{newRetiredGoogleDriveSettings(), newRetiredRcloneSettings()},
		})
	})
}
