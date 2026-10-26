// Package notification holds what every part of the module says about a notification: what it is
// about, its text and the type of notifier it goes to.
package notification

// Category says what a notification is about; a webhook notifier can accept only some categories.
type Category string

const (
	// CategoryAll matches every category: on the accepted list of a webhook it accepts every
	// notification, and a test notification carries it, so a test always goes out.
	CategoryAll                 Category = "ALL"
	CategoryBackupSuccess       Category = "BACKUP_SUCCESS"
	CategoryBackupFailed        Category = "BACKUP_FAILED"
	CategoryHealthcheckSuccess  Category = "HEALTHCHECK_SUCCESS"
	CategoryHealthcheckFailed   Category = "HEALTHCHECK_FAILED"
	CategoryVerificationSuccess Category = "VERIFICATION_SUCCESS"
	CategoryVerificationFailed  Category = "VERIFICATION_FAILED"
)

// Notification is a message about an event of a database, sent to the notifiers of that database.
type Notification struct {
	Category Category
	Heading  string
	Message  string
}
