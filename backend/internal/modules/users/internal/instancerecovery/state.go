// Package instancerecovery is what recovery of the instance needs from the users tables: whether a
// startup recovery target holds accounts or changed settings, and the end of every session and
// code a recovered snapshot carries.
package instancerecovery

// State is what the users tables of a recovery target hold.
type State struct {
	// HasAccounts is true for any user row, a sole root account and an invitation included.
	HasAccounts bool
	// HasChangedSettings is true when the instance settings differ from those a new instance
	// starts with.
	HasChangedSettings bool
	// AutomaticTables are populated tables whose rows the application created by itself.
	AutomaticTables []string
}
