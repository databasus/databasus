package snapshot

import "errors"

var ErrInsufficientDiskSpace = errors.New("the temporary folder has too little free space for the Databasus backup")
