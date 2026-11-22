package shared

import "strings"

const neonDomain = "neon.tech"

// IsNeonHost judges the host a user configured, never a tunnel address: the host names the Neon
// project the user pays for, wherever the connection is routed.
func IsNeonHost(host string) bool {
	normalizedHost := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")

	return normalizedHost == neonDomain || strings.HasSuffix(normalizedHost, "."+neonDomain)
}
