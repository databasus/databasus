package restore

import "strings"

// The runner uses this to reclassify a non-zero pg_restore exit away from
// BackupRejected: exceeding an estimate-derived bound is agent infra, not
// proof the backup is corrupt.
func IsDiskExhausted(stderrTail string) bool {
	lowered := strings.ToLower(stderrTail)

	for _, sig := range []string{
		"no space left on device",
		"could not extend",
		"disk full",
		"wrote only",
	} {
		if strings.Contains(lowered, sig) {
			return true
		}
	}

	return false
}
