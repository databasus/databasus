// Package files owns the obligation to remove a stored backup file. The obligation is created before
// the first byte reaches the provider and released only by the transaction that publishes the file as
// part of a backup, so cleanup never depends on a caller remembering to delete anything.
package files
