package containers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetUnavailableImageReason(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name          string
		architecture  string
		image         string
		isUnavailable bool
	}{
		{"excludes mysql 5.7 when running on arm64", "arm64", "mysql:5.7", true},
		{"excludes mariadb 5.5 when running on arm64", "arm64", "mariadb:5.5", true},
		{"normalizes arm when the kernel reports aarch64", "aarch64", "mysql:5.7", true},
		{"includes mysql 5.7 when running on amd64", "amd64", "mysql:5.7", false},
		{"includes mariadb 5.5 when running on amd64", "amd64", "mariadb:5.5", false},
		{"includes old mysql when the kernel reports x86_64", "x86_64", "mysql:5.7", false},
		{"includes supported mysql when running on arm64", "arm64", "mysql:8.0", false},
		{"includes supported mariadb when running on arm64", "arm64", "mariadb:10.1", false},
		{"includes unknown images when no exclusion exists", "arm64", "mysql:missing", false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			reason := getUnavailableImageReason(
				imageAvailabilitySpec{Architecture: scenario.architecture, Image: scenario.image},
			)

			assert.Equal(t, scenario.isUnavailable, reason != "")
		})
	}
}
