package googledrive

import "errors"

// A missing backups folder means every file in it is missing too, so deletion
// treats it as a confirmed absence rather than a failure to retry.
var errBackupsFolderNotFound = errors.New("databasus_backups folder not found")
