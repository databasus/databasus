package system_version

import "os"

const defaultVersion = "3.26.0"

// GetAppVersion returns the current application version, falling back to a
// hardcoded default when APP_VERSION is unset.
func GetAppVersion() string {
	version := os.Getenv("APP_VERSION")
	if version == "" {
		return defaultVersion
	}

	return version
}
