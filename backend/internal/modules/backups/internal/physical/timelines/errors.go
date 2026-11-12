package timelines

import "errors"

var ErrHistoryPublicationMissing = errors.New("concurrent history publication is no longer available")
