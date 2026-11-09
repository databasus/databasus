package upgrade

import "testing"

const (
	rcloneReleaseImage     = "databasus/databasus:v3.67.1@sha256:663dd65f995d31802b41d4eb20399798285bff7367cc4f91dcb5617f6af83b80"
	retirementRcloneSecret = "retirement-synthetic-rclone-secret"
)

type previousRcloneRequest struct {
	Type          string `json:"type"`
	ConfigContent string `json:"configContent"`
	RemotePath    string `json:"remotePath"`
}

func TestRcloneRetirement(t *testing.T) {
	t.Parallel()
	t.Run("preserves 3.67.1 on refusal and upgrades after removal through its API", func(t *testing.T) {
		t.Parallel()
		runRetirementScenario(t, retirementScenario{
			image:    rcloneReleaseImage,
			guidance: "Databasus 3.67.1 supports rclone storages. Roll back to Databasus 3.67.1 and remove all rclone storages before upgrading, or continue using 3.67.1.",
			settings: []any{newRetiredRcloneSettings()},
		})
	})
}

func newRetiredRcloneSettings() previousRcloneRequest {
	return previousRcloneRequest{
		Type:          "RCLONE",
		ConfigContent: "[retirement]\ntype = s3\nprovider = Minio\naccess_key_id = synthetic\nsecret_access_key = " + retirementRcloneSecret + "\nendpoint = https://retirement.invalid\n",
		RemotePath:    "retirement/backups",
	}
}
