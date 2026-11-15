package local

import "errors"

var (
	ErrProbeSaveFailed   = errors.New("save a file through local storage")
	ErrProbeDeleteFailed = errors.New("delete a file through local storage")
	// ErrBackupFolderProbeFailed is a recovered local storage whose backup folder this host cannot
	// write, read back or clean.
	ErrBackupFolderProbeFailed = errors.New("the backup folder is not usable on this host")
)
