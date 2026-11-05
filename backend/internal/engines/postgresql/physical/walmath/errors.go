package walmath

import "fmt"

type NotWalFilenameError struct {
	Filename string
}

func (e NotWalFilenameError) Error() string {
	return fmt.Sprintf("walmath: not a WAL filename: %q", e.Filename)
}

type IncorrectLogSegNoError struct {
	Filename string
}

func (e IncorrectLogSegNoError) Error() string {
	return fmt.Sprintf("walmath: incorrect log seg no in WAL filename: %q", e.Filename)
}
